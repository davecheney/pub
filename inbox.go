package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/mastodon"
	"gorm.io/gorm"
)

type IndexCmd struct {
}

func (i *IndexCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	var activities []activitypub.Activity
	if err := db.Where("processed_at IS NULL").Find(&activities).Error; err != nil {
		return err
	}

	ip := &inboxProcessor{
		db:       db,
		accounts: mastodon.NewAccounts(db),
	}

	for _, activity := range activities {
		if err := ip.Process(&activity); err != nil {
			fmt.Println(err)
			continue
		}
		t := time.Now()
		activity.ProcessedAt = &t
		if err := db.Save(&activity).Error; err != nil {
			return err
		}

	}

	return nil
}

type inboxProcessor struct {
	db       *gorm.DB
	accounts *mastodon.Accounts
}

func (ip *inboxProcessor) Process(activity *activitypub.Activity) error {
	act := activity.Activity
	id, _ := act["id"].(string)
	typ, _ := act["type"].(string)
	actorID, _ := act["actor"].(string)
	fmt.Println("process: id:", id, "type:", typ, "actor:", actorID)

	account, err := ip.accounts.FindOrCreateAccount(actorID)
	if err != nil {
		return err
	}
	switch typ {
	case "Create":
		create := act["object"].(map[string]any)
		return ip.processCreate(account, create)
	default:
		return nil
	}
}

func (ip *inboxProcessor) processCreate(account *mastodon.Account, obj map[string]any) error {
	// json.MarshalOptions{}.MarshalFull(json.EncodeOptions{Indent: "  "}, os.Stdout, obj)
	// fmt.Println()
	typ, _ := obj["type"].(string)
	switch typ {
	case "Note":
		return ip.processCreateNote(account, obj)
	default:
		return nil
	}
}

func (ip *inboxProcessor) processCreateNote(account *mastodon.Account, obj map[string]any) error {
	published, err := timeFromAny(obj["published"])
	if err != nil {
		return err
	}

	status := mastodon.Status{
		Model: gorm.Model{
			CreatedAt: published,
		},
		AccountID: account.ID,
		// InReplyToID        *uint
		// InReplyToAccountID *uint
		Sensitive:   boolFromAny(obj["sensitive"]),
		SpoilerText: stringFromAny(obj["summary"]),
		Visibility:  "public",
		Language:    "en",
		Content:     stringFromAny(obj["content"]),
	}

	if err := ip.db.Create(&status).Error; err != nil {
		return err
	}
	return nil
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func timeFromAny(v any) (time.Time, error) {
	switch v := v.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	case time.Time:
		return v, nil
	default:
		return time.Time{}, errors.New("timeFromAny: invalid type")
	}
}
