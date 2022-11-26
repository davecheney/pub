package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
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
	if err := db.Find(&activities).Error; err != nil {
		return err
	}

	ip := &inboxProcessor{
		db:       db,
		accounts: mastodon.NewAccounts(db),
		statuses: mastodon.NewStatuses(db),
	}

	for _, activity := range activities {
		if err := ip.Process(&activity); err != nil {
			fmt.Println(err)
			continue
		}
		if err := db.Delete(&activity).Error; err != nil {
			return err
		}
	}

	return nil
}

type inboxProcessor struct {
	db       *gorm.DB
	accounts *mastodon.Accounts
	statuses *mastodon.Statuses
}

func (ip *inboxProcessor) Process(activity *activitypub.Activity) error {
	act := activity.Activity
	id := stringFromAny(act["id"])
	typ := stringFromAny(act["type"])
	actor := stringFromAny(act["actor"])
	fmt.Println("process: id:", id, "type:", typ, "actor:", actor)
	switch typ {
	case "Create":
		create := mapFromAny(act["object"])
		return ip.processCreate(create)
	default:
		return nil
	}
}

func (ip *inboxProcessor) processCreate(obj map[string]any) error {
	typ := stringFromAny(obj["type"])
	switch typ {
	case "Note":
		return ip.processCreateNote(obj)
	default:
		return nil
	}
}

func (ip *inboxProcessor) processCreateNote(obj map[string]any) error {
	account, err := ip.accounts.FindOrCreateAccount(stringFromAny(obj["attributedTo"]))
	if err != nil {
		return err
	}

	published, err := timeFromAny(obj["published"])
	if err != nil {
		return err
	}

	var inReplyTo *mastodon.Status
	if inReplyToAtomUri, ok := obj["inReplyTo"].(string); ok {
		inReplyTo, err = ip.statuses.FindOrCreateStatus(inReplyToAtomUri)
		if err != nil {
			return err
		}
	}
	status := mastodon.Status{
		Model: gorm.Model{
			CreatedAt: published,
		},
		AccountID: account.ID,
		Account:   account,
		URI:       stringFromAny(obj["atomUri"]),
		InReplyToID: func() *uint {
			if inReplyTo != nil {
				return &inReplyTo.ID
			}
			return nil
		}(),
		InReplyToAccountID: func() *uint {
			if inReplyTo != nil {
				return &inReplyTo.AccountID
			}
			return nil
		}(),
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

func actorFromStatusURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	parts := strings.Split(u.Path, "/")
	if len(parts) < 3 {
		return "", errors.New("actorFromStatusURI: invalid path")
	}
	return fmt.Sprintf("%s://%s/users/%s", u.Scheme, u.Host, parts[2]), nil
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

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}
