package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/m"
	"gorm.io/gorm"
)

type InboxCmd struct {
	Domain string `required:"" help:"domain name of the instance"`
}

func (i *InboxCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	svc, err := m.NewService(db, i.Domain)
	if err != nil {
		return err
	}

	var activities []activitypub.Activity
	if err := db.Find(&activities).Error; err != nil {
		return err
	}

	ip := &inboxProcessor{
		db:      db,
		service: svc,
	}

	for i := range activities {
		if err := ip.Process(&activities[i]); err != nil {
			fmt.Println(err)
			continue
		}
		if err := db.Delete(&activities[i]).Error; err != nil {
			return err
		}
	}

	return nil
}

type inboxProcessor struct {
	db      *gorm.DB
	service *m.Service
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
	account, err := ip.service.Accounts().FindOrCreateAccount(stringFromAny(obj["attributedTo"]))
	if err != nil {
		return err
	}

	published, err := timeFromAny(obj["published"])
	if err != nil {
		return err
	}

	var visibility string
	for _, recipient := range obj["to"].([]interface{}) {
		if recipient == "https://www.w3.org/ns/activitystreams#Public" {
			visibility = "public"
			break
		}
	}
	switch visibility {
	case "public":
		// cool
	default:
		return fmt.Errorf("unsupported visibility %q", visibility)
	}

	var inReplyTo *m.Status
	if inReplyToAtomUri, ok := obj["inReplyTo"].(string); ok {
		inReplyTo, err = ip.service.Statuses().FindOrCreateStatus(inReplyToAtomUri)
		if err != nil {
			return err
		}
	}

	conversationID := uint(0)
	if inReplyTo != nil {
		conversationID = inReplyTo.ConversationID
	} else {
		conv := m.Conversation{
			Visibility: visibility,
		}
		if err := ip.db.Create(&conv).Error; err != nil {
			return err
		}
		conversationID = conv.ID
	}

	status := m.Status{
		Model: gorm.Model{
			CreatedAt: published,
		},
		AccountID:      account.ID,
		Account:        account,
		ConversationID: conversationID,
		URI:            stringFromAny(obj["atomUri"]),
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
