package models

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/pub/internal/activitypub"
	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
)

// Object represents an ActivityPub object.
type Object struct {
	ID         snowflake.ID   `gorm:"primarykey;autoIncrement:false"`
	Type       string         `gorm:"type:varchar(16);not null"`
	URI        string         `gorm:"type:varchar(255);not null;uniqueIndex"`
	Properties map[string]any `gorm:"serializer:json;not null"`
}

func (o *Object) BeforeSave(tx *gorm.DB) error {
	if o.URI == "" {
		uri, ok := o.Properties["id"].(string)
		if !ok {
			return errors.New("object has no id")
		}
		o.URI = uri
	}

	if o.Type == "" {
		typ, ok := o.Properties["type"].(string)
		if !ok {
			return fmt.Errorf("object %s has no type", o.URI)
		}
		o.Type = typ
	}

	if o.ID == 0 {
		switch published := o.Properties["published"].(type) {
		case string:
			publishedAt, err := time.Parse(time.RFC3339, published)
			if err != nil {
				return fmt.Errorf("object %s, %s has invalid published date %q: %w", o.URI, o.Type, published, err)
			}
			o.ID = snowflake.TimeToID(publishedAt)
		default:
			o.ID = snowflake.Now()
		}
	}

	if _, err := url.Parse(o.URI); err != nil {
		return fmt.Errorf("object has invalid uri %q: %w", o.URI, err)
	}
	if o.ID == 0 {
		return fmt.Errorf("object %s has empty id", o.URI)
	}
	if o.Type == "" {
		return fmt.Errorf("object %s has empty type", o.URI)
	}

	fmt.Println("BeforeSave:", "id:", o.ID, "type:", o.Type, "uri:", o.URI)
	return nil
}

func (o *Object) AfterSave(tx *gorm.DB) error {
	// fmt.Println("AfterSave:", "id:", o.ID, "type:", o.Type, "uri:", o.URI)
	switch o.Type {
	case "Person", "Service":
		return o.maybeSaveActor(tx)
	case "Note", "Question":
		return o.maybeCreateStatus(tx)
	case "Announce":
		return o.maybeCreateReblog(tx)
	default:
		return nil
	}
}

// maybeSaveActor updates the models.Actor table with the object's properties iff
// the object is a Person or Service.
func (o *Object) maybeSaveActor(tx *gorm.DB) error {
	u, err := url.Parse(o.URI)
	if err != nil {
		return err
	}
	a := &Actor{
		ObjectID: o.ID,
		Type:     ActorType(o.Type),
		Name:     stringFromAny(o.Properties["preferredUsername"]),
		Domain:   u.Host,
	}
	return tx.Save(a).Error
}

type ActorAttachment struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (o *Object) maybeCreateStatus(tx *gorm.DB) error {
	// fmt.Println("maybeCreateStatus:", "id:", o.ID, "type:", o.Type, "uri:", o.URI)
	attributedTo, ok := o.Properties["attributedTo"].(string)
	if !ok {
		return fmt.Errorf("object %s has no attributedTo", o.URI)
	}

	actor, err := NewActors(tx).FindOrCreateByURI(attributedTo)
	if err != nil {
		return fmt.Errorf("failed to find actor %s: %w", attributedTo, err)
	}

	conv := &Conversation{
		Visibility: Visibility(visiblity(o.Properties)),
	}
	var inReplyTo *Status
	if replyTo, ok := o.Properties["inReplyTo"].(string); ok {
		inReplyTo, err = NewStatuses(tx).FindOrCreateByURI(replyTo)
		if err != nil {
			switch {
			case requests.HasStatusErr(err, http.StatusNotFound, http.StatusGone, http.StatusUnauthorized):
				// if the inReplyTo status doesn't exist or isn't visible,
				// we can still create the status, but we won't be able to
				// set the conversation correctly.
				inReplyTo = nil
			default:
				return err
			}
		}
	}

	if inReplyTo != nil {
		conv = inReplyTo.Conversation
	}

	var updatedAt time.Time
	if updated, ok := o.Properties["updated"].(string); ok {
		if updatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
			return fmt.Errorf("object %s has invalid updated date %s: %w", o.URI, updated, err)
		}
	}

	status := Status{
		ObjectID:         o.ID,
		UpdatedAt:        updatedAt,
		ActorID:          actor.ObjectID,
		Actor:            actor,
		Conversation:     conv,
		Visibility:       conv.Visibility,
		InReplyToID:      inReplyToID(inReplyTo),
		InReplyToActorID: inReplyToActorID(inReplyTo),
	}

	return tx.Save(&status).Error
}

func (o *Object) maybeCreateReblog(tx *gorm.DB) error {
	target, ok := o.Properties["object"].(string)
	if !ok {
		return fmt.Errorf("object %s has no target", o.URI)
	}
	original, err := NewStatuses(tx).FindOrCreateByURI(target)
	if err != nil {
		return err
	}
	actor, err := NewActors(tx).FindOrCreateByURI(stringFromAny(o.Properties["actor"]))
	if err != nil {
		return err
	}

	var updatedAt time.Time
	if updated, ok := o.Properties["updated"].(string); ok {
		if updatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
			return fmt.Errorf("object %s has invalid updated date %s: %w", o.URI, updated, err)
		}
	}

	conv := &Conversation{
		Visibility: original.Visibility,
	}

	status := &Status{
		ObjectID:     o.ID,
		UpdatedAt:    updatedAt,
		ActorID:      actor.ObjectID,
		Actor:        actor,
		Conversation: conv,
		Visibility:   conv.Visibility,
		ReblogID:     &original.ObjectID,
		Reblog:       original,
	}

	return tx.Save(status).Error
}

func inReplyToID(inReplyTo *Status) *snowflake.ID {
	if inReplyTo != nil {
		return &inReplyTo.ObjectID
	}
	return nil
}

func inReplyToActorID(inReplyTo *Status) *snowflake.ID {
	if inReplyTo != nil {
		return &inReplyTo.ActorID
	}
	return nil
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
	return "direct" // hack
}

func anyToSlice(v any) []any {
	switch v := v.(type) {
	case []any:
		return v
	default:
		return nil
	}
}

func fetchObject(ctx context.Context, uri string) (map[string]any, error) {
	client, ok := activitypub.FromContext(ctx)
	if !ok {
		return nil, errors.New("no activitypub client in context")
	}
	var obj map[string]any
	err := client.Fetch(ctx, uri, &obj)
	// fmt.Println("fetched object:", "id", uri, "type", obj["type"], "error", err)
	return obj, err
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}
