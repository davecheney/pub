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
	case "Announce":
		return ip.processAnnounce(act)
	case "Undo":
		undo := mapFromAny(act["object"])
		return ip.processUndo(undo)
	case "Add":
		return ip.processAdd(act)
	case "Remove":
		return ip.processRemove(act)
	default:
		return fmt.Errorf("unknown activity type: %q", typ)
	}
}

func (ip *inboxProcessor) processAdd(act map[string]any) error {
	target := stringFromAny(act["target"])
	switch target {
	case stringFromAny(act["actor"]) + "/collections/featured":
		status, err := ip.service.Statuses().FindByURI(stringFromAny(act["object"]))
		if err != nil {
			return err
		}
		status.Pinned = true
		return ip.db.Save(status).Error
	default:
		x, _ := json.MarshalIndent(act, "", "  ")
		fmt.Println("processAdd:", string(x))
		return errors.New("not implemented")
	}
}

func (ip *inboxProcessor) processRemove(act map[string]any) error {
	target := stringFromAny(act["target"])
	switch target {
	case stringFromAny(act["actor"]) + "/collections/featured":
		status, err := ip.service.Statuses().FindByURI(stringFromAny(act["object"]))
		if err != nil {
			return err
		}
		status.Pinned = false
		return ip.db.Save(status).Error
	default:
		x, _ := json.MarshalIndent(act, "", "  ")
		fmt.Println("processRemove:", string(x))
		return errors.New("not implemented")
	}
}

func (ip *inboxProcessor) processUndo(obj map[string]any) error {
	typ := stringFromAny(obj["type"])
	switch typ {
	case "Announce":
		return ip.processUndoAnnounce(obj)
	default:
		return fmt.Errorf("unknown undo object type: %q", typ)
	}
}

func (ip *inboxProcessor) processUndoAnnounce(obj map[string]any) error {
	id := stringFromAny(obj["id"])
	status, err := ip.service.Statuses().FindByURI(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// already deleted
		return nil
	}
	if err != nil {
		return err
	}
	return ip.db.Delete(status).Error
}

func (ip *inboxProcessor) processCreate(obj map[string]any) error {
	typ := stringFromAny(obj["type"])
	switch typ {
	case "Note":
		return ip.processCreateNote(obj)
	case "Question":
		return ip.processCreateQuestion(obj)
	default:
		return fmt.Errorf("unknown create object type: %q", typ)
	}
}

func (ip *inboxProcessor) processCreateQuestion(obj map[string]any) error {
	x, _ := json.MarshalIndent(obj, "", "  ")
	fmt.Println("processCreateQuestion:", string(x))
	return errors.New("not implemented")
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
		vis := visiblity(obj)
		if vis == "" {
			x, _ := json.MarshalIndent(obj, "", "  ")
			return nil, fmt.Errorf("unsupported visibility %q: %s", vis, x)
		}

		var inReplyTo *m.Status
		if inReplyToAtomUri, ok := obj["inReplyTo"].(string); ok {
			remoteStatusFetcher := ip.service.Statuses().NewRemoteStatusFetcher()
			inReplyTo, err = ip.service.Statuses().FindOrCreate(inReplyToAtomUri, remoteStatusFetcher.Fetch)
			if err != nil {
				fmt.Println("inReplyToAtomUri:", inReplyToAtomUri, "err:", err)
			}
		}

		conversationID := uint32(0)
		if inReplyTo != nil {
			conversationID = inReplyTo.ConversationID
		} else {
			conv := m.Conversation{
				Visibility: vis,
			}
			if err := ip.db.Create(&conv).Error; err != nil {
				return nil, err
			}
			conversationID = conv.ID
		}

		att, _ := json.MarshalIndent(obj["attachment"], "", "  ")
		fmt.Println("attachment:", string(att))

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

func (ip *inboxProcessor) processAnnounce(obj map[string]any) error {
	target := stringFromAny(obj["object"])
	original, err := ip.service.Statuses().FindOrCreate(target, ip.service.Statuses().NewRemoteStatusFetcher().Fetch)
	if err != nil {
		return err
	}

	actor, err := ip.service.Actors().FindOrCreate(stringFromAny(obj["actor"]), ip.service.Actors().NewRemoteActorFetcher().Fetch)
	if err != nil {
		return err
	}

	published, err := timeFromAny(obj["published"])
	if err != nil {
		return err
	}

	conv := m.Conversation{
		Visibility: "public",
	}
	if err := ip.db.Create(&conv).Error; err != nil {
		return err
	}

	status := &m.Status{
		ID:               snowflake.TimeToID(published),
		ActorID:          actor.ID,
		Actor:            actor,
		ConversationID:   conv.ID,
		URI:              stringFromAny(obj["id"]),
		InReplyToID:      nil,
		InReplyToActorID: nil,
		Sensitive:        false,
		SpoilerText:      "",
		Visibility:       "public",
		Language:         "",
		Note:             "",
		ReblogID:         &original.ID,
		Reblog:           original,
	}

	return ip.db.Create(status).Error
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

func visiblity(obj map[string]any) string {
	actor := stringFromAny(obj["attributedTo"])
	for _, recipient := range anyToSlice(obj["to"]) {
		switch recipient {
		case "https://www.w3.org/ns/activitystreams#Public":
			return "public"
		case actor + "/followers":
			return "limited"
		}
	}
	for _, recipient := range anyToSlice(obj["cc"]) {
		switch recipient {
		case "https://www.w3.org/ns/activitystreams#Public":
			return "public"
		}
	}
	return ""
}
