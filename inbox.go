package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-json-experiment/json"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/mastodon"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type IndexCmd struct {
	DSN string `help:"data source name"`
}

func (i *IndexCmd) Run(ctx *Context) error {
	dsn := i.DSN + "?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &ctx.Config)
	if err != nil {
		return err
	}

	var activities []activitypub.Activity
	if err := db.Find(&activities).Where("processed_at IS NULL").Error; err != nil {
		return err
	}

	ip := &inboxProcessor{
		db: db,
	}

	for _, activity := range activities {
		if err := ip.Process(&activity); err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

type inboxProcessor struct {
	db *gorm.DB
}

func (ip *inboxProcessor) Process(activity *activitypub.Activity) error {
	var act map[string]any
	r := strings.NewReader(activity.Activity)
	if err := json.UnmarshalFull(r, &act); err != nil {
		return err
	}
	id, _ := act["id"].(string)
	typ, _ := act["type"].(string)
	actorID, _ := act["actor"].(string)
	fmt.Println("process: id:", id, "type:", typ, "actor:", actorID)
	// json.MarshalOptions{}.MarshalFull(json.EncodeOptions{Indent: "  "}, os.Stdout, act)
	// fmt.Println()
	actor, err := ip.fetchActor(actorID)
	if err != nil {
		return err
	}
	if err := ip.maybeCreateAccount(actor); err != nil {
		return err
	}
	return nil
}

func (ip *inboxProcessor) maybeCreateAccount(actor *activitypub.Actor) error {
	var account mastodon.Account
	err := ip.db.First(&account, "username = ? AND domain = ?", actor.Username(), actor.Domain()).Error
	if err == nil {
		// found cached key
		return nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var obj map[string]any
	if err := json.UnmarshalFull(bytes.NewReader(actor.Object), &obj); err != nil {
		return err
	}
	account = mastodon.Account{
		Username:    actor.Username(),
		Domain:      actor.Domain(),
		Acct:        actor.Username() + "@" + actor.Domain(),
		DisplayName: obj["name"].(string),
		Locked:      false,
		Bot:         false,
		Note:        obj["summary"].(string),
		URL:         obj["url"].(string),
	}

	if err := ip.db.Create(&account).Error; err != nil {
		return err
	}

	// json.MarshalOptions{}.MarshalFull(json.EncodeOptions{Indent: "  "}, os.Stdout, actor.Object)
	// fmt.Println()

	return nil
}

func (ip *inboxProcessor) fetchActor(id string) (*activitypub.Actor, error) {
	var actor activitypub.Actor
	err := ip.db.Where("actor_id = ?", id).First(&actor).Error
	if err == nil {
		// found cached key
		return &actor, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	req, err := http.NewRequest("GET", id, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: newrequest: %w", err)
	}
	req.Header.Set("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: %v do: %w", req.URL, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchActor: %v status: %d", resp.Request.URL, resp.StatusCode)
	}
	var v map[string]interface{}
	if err := json.UnmarshalFull(resp.Body, &v); err != nil {
		return nil, fmt.Errorf("fetchActor: jsondecode: %w", err)
	}
	obj, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: jsonencode: %w", err)
	}
	actor = activitypub.Actor{
		ActorID:   id,
		Type:      v["type"].(string),
		Object:    obj,
		PublicKey: v["publicKey"].(map[string]interface{})["publicKeyPem"].(string),
	}
	if err := ip.db.Create(&actor).Error; err != nil {
		return nil, fmt.Errorf("fetchActor: create: %w", err)
	}
	return &actor, nil
}
