package models

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/pub/internal/httpsig"
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

func (o *Object) BeforeCreate(tx *gorm.DB) error {
	return forEach(tx, o.maybeSetURI, o.maybeSetID, o.maybeSetType)
}

func (o *Object) maybeSetURI(tx *gorm.DB) error {
	if o.URI == "" {
		uri, ok := o.Properties["id"].(string)
		if !ok {
			return errors.New("object has no id")
		}
		o.URI = uri
	}
	return nil
}

func (o *Object) maybeSetID(tx *gorm.DB) error {
	if o.ID == 0 {
		published, ok := o.Properties["published"].(string)
		switch o.Type {
		case "Person", "Service":
			if !ok || published == "" {
				// misskey/calckey don't seem to set published for actors, so set the ID to now.
				o.ID = snowflake.Now()
				return nil
			}
			fallthrough
		default:
			publishedAt, err := time.Parse(time.RFC3339, published)
			if err != nil {
				return fmt.Errorf("object %s has invalid published date %s: %w", o.URI, published, err)
			}
			o.ID = snowflake.TimeToID(publishedAt)
		}
	}
	return nil
}

func (o *Object) maybeSetType(tx *gorm.DB) error {
	if o.Type == "" {
		typ, ok := o.Properties["type"].(string)
		if !ok {
			return fmt.Errorf("object %s has no type", o.URI)
		}
		o.Type = typ
	}
	return nil
}

func (o *Object) AfterCreate(tx *gorm.DB) error {
	switch o.Type {
	case "Person", "Service":
		return o.maybeSaveActor(tx)
	case "Note", "Question":
		// return o.maybeFetchActor(tx)
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

type ActorAttribute struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (o *Object) maybeCreateStatus(tx *gorm.DB) error {
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
			return err
		}
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

	// updatedAt := act.Updated
	// if updatedAt.IsZero() {
	// 	updatedAt = obj.ID.ToTime()
	// }

	conv := &Conversation{
		Visibility: Visibility(visiblity(o.Properties)),
	}

	status := &Status{
		ObjectID: o.ID,
		// UpdatedAt: updatedAt,
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

// maybeFetchActor fetches the object's actor and creates it if it doesn't exist.
func (o *Object) maybeFetchActor(tx *gorm.DB) error {
	attributedTo, ok := o.Properties["attributedTo"].(string)
	if !ok {
		return fmt.Errorf("object %s has no attributedTo property", o.URI)
	}
	var actor []Object
	if err := tx.Where("uri = ?", attributedTo).Find(&actor).Error; err != nil {
		return fmt.Errorf("failed to find actor %s: %w", attributedTo, err)
	}
	if len(actor) > 0 {
		return nil
	}

	if err := o.fetchActor(tx, attributedTo); err != nil {
		return fmt.Errorf("failed to fetch actor %s: %w", attributedTo, err)
	}
	return nil
}

func (o *Object) fetchActor(tx *gorm.DB, uri string) error {
	attributedTo, ok := o.Properties["attributedTo"].(string)
	if !ok {
		return fmt.Errorf("object %s has no attributedTo property", o.URI)
	}
	obj, err := fetchObject(tx.Statement.Context, attributedTo)
	if err != nil {
		return err
	}
	return tx.Create(&Object{Properties: obj}).Error
}

func fetchObject(ctx context.Context, uri string) (map[string]any, error) {
	instance, ok := ctx.Value("instance").(*Instance)
	if !ok {
		return nil, fmt.Errorf("no instance in context, got %T", ctx.Value("instance"))
	}

	fmt.Println("fetching object:", "id", uri)
	client, err := NewClient(instance.Admin)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	return obj, client.Fetch(ctx, uri, &obj)
}

// Client is an ActivityPub client which can be used to fetch remote
// ActivityPub resources.
type Client struct {
	keyID      string
	privateKey crypto.PrivateKey
}

// NewClient returns a new ActivityPub client.
func NewClient(signAs *Account) (*Client, error) {
	privPem, _ := pem.Decode(signAs.PrivateKey)
	if privPem == nil || privPem.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("expected RSA PRIVATE KEY")
	}

	var parsedKey interface{}
	var err error
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPem.Bytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPem.Bytes); err != nil { // note this returns type `interface{}`
			return nil, err
		}
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("expected *rsa.PrivateKey")
	}

	return &Client{
		keyID:      signAs.Actor.PublicKeyID(),
		privateKey: privateKey,
	}, nil
}

// Fetch fetches the ActivityPub resource at the given URL and decodes it into the given object.
func (c *Client) Fetch(ctx context.Context, uri string, obj interface{}) error {
	return requests.URL(uri).
		Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
		Transport(c).
		CheckContentType("application/ld+json", "application/activity+json", "application/json").
		CheckStatus(http.StatusOK).
		ToJSON(obj).
		Fetch(ctx)
}

func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := httpsig.Sign(req, c.keyID, c.privateKey, nil); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}
