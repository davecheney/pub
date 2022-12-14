package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/internal/snowflake"
	"github.com/davecheney/m/m"
	"gorm.io/gorm"
)

type InboxCmd struct {
}

func (i *InboxCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	var activities []activitypub.Activity
	if err := db.Find(&activities).Error; err != nil {
		return err
	}

	ip := &inboxProcessor{
		db:      db,
		service: m.NewService(db),
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
	act := activity.Object
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
	uri := stringFromAny(obj["atomUri"]) // TODO should be using URL and going from webfinger to account
	if uri == "" {
		uri = stringFromAny(obj["id"])
	}
	_, err := ip.service.Statuses().FindOrCreate(uri, func(string) (*m.Status, error) {
		fetcher := ip.service.Actors().NewRemoteActorFetcher()
		actor, err := ip.service.Actors().FindOrCreate(stringFromAny(obj["attributedTo"]), fetcher.Fetch)
		if err != nil {
			return nil, err
		}

		published, err := timeFromAny(obj["published"])
		if err != nil {
			return nil, err
		}

		var visibility string
		for _, recipient := range anyToSlice(obj["to"]) {
			switch recipient {
			case "https://www.w3.org/ns/activitystreams#Public":
				visibility = "public"
			case actor.ToAcct().Followers():
				visibility = "limited"
			}
		}
		if visibility == "" {
			x, _ := json.MarshalIndent(obj, "", "  ")
			return nil, fmt.Errorf("unsupported visibility %q: %s", visibility, x)
		}

		var inReplyTo *m.Status
		if inReplyToAtomUri, ok := obj["inReplyTo"].(string); ok {
			remoteStatusFetcher := ip.service.Statuses().NewRemoteStatusFetcher()
			inReplyTo, err = ip.service.Statuses().FindOrCreate(inReplyToAtomUri, remoteStatusFetcher.Fetch)
			if err != nil {
				return nil, err
			}
		}

		conversationID := uint32(0)
		if inReplyTo != nil {
			conversationID = inReplyTo.ConversationID
		} else {
			conv := m.Conversation{
				Visibility: visibility,
			}
			if err := ip.db.Create(&conv).Error; err != nil {
				return nil, err
			}
			conversationID = conv.ID
		}

		return &m.Status{
			ID:             snowflake.TimeToID(published),
			ActorID:        actor.ID,
			Actor:          actor,
			ConversationID: conversationID,
			URI:            uri,
			InReplyToID: func() *uint64 {
				if inReplyTo != nil {
					return &inReplyTo.ID
				}
				return nil
			}(),
			InReplyToActorID: func() *uint64 {
				if inReplyTo != nil {
					return &inReplyTo.ActorID
				}
				return nil
			}(),
			Sensitive:   boolFromAny(obj["sensitive"]),
			SpoilerText: stringFromAny(obj["summary"]),
			Visibility:  "public",
			Language:    "en",
			Note:        stringFromAny(obj["content"]),
		}, nil
	})
	return err
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

func anyToSlice(v any) []any {
	switch v := v.(type) {
	case []any:
		return v
	default:
		return nil
	}
}
