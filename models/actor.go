package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/snowflake"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type Actor struct {
	ObjectID       snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Object         *ActorObject `gorm:"constraint:OnDelete:CASCADE;<-:false"`
	UpdatedAt      time.Time    `gorm:"autoUpdateTime:false"`
	Type           ActorType    `gorm:"default:'Person';not null"`
	Name           string       `gorm:"size:64;uniqueIndex:idx_actor_name_domain;not null"`
	Domain         string       `gorm:"size:64;uniqueIndex:idx_actor_name_domain;not null"`
	FollowersCount int32        `gorm:"default:0;not null"`
	FollowingCount int32        `gorm:"default:0;not null"`
	StatusesCount  int32        `gorm:"default:0;not null"`
	LastStatusAt   time.Time
}

type ActorObject struct {
	ID         snowflake.ID
	Type       string
	URI        string
	Properties struct {
		Type string `json:"type"`
		// The Actor's unique global identifier.
		ID                string `json:"id"`
		Inbox             string `json:"inbox"`
		Outbox            string `json:"outbox"`
		PreferredUsername string `json:"preferredUsername"`
		Name              string `json:"name"`
		Summary           string `json:"summary"`
		Icon              struct {
			Type      string `json:"type"`
			MediaType string `json:"mediaType"`
			URL       string `json:"url"`
		} `json:"icon"`
		Image struct {
			Type      string `json:"type"`
			MediaType string `json:"mediaType"`
			URL       string `json:"url"`
		} `json:"image"`
		Endpoints struct {
			SharedInbox string `json:"sharedInbox"`
		} `json:"endpoints"`
		ManuallyApprovesFollowers bool `json:"manuallyApprovesFollowers"`
		PublicKey                 struct {
			ID           string `json:"id"`
			Owner        string `json:"owner"`
			PublicKeyPem string `json:"publicKeyPem"`
		} `json:"publicKey"`
		Attachments []ActorAttachment `json:"attachment"`
	} `gorm:"serializer:json;not null"`
}

func (ActorObject) TableName() string {
	return "objects"
}

type ActorType string

func (ActorType) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "postgres":
		return "enum('Person', 'Application', 'Service', 'Group', 'Organization', 'LocalPerson', 'LocalService')"
	case "sqlite":
		return "TEXT"
	default:
		return ""
	}
}

// Inbox returns the actor's inbox URL, or shared inbox URL if applicable.
func (a *Actor) Inbox() string {
	if a.SharedInboxURL() != "" {
		return a.SharedInboxURL()
	}
	return a.InboxURL()
}

func (a *Actor) Attributes() []ActorAttachment {
	return a.Object.Properties.Attachments
}

func (a *Actor) Avatar() string {
	return a.Object.Properties.Icon.URL
}

func (a *Actor) DisplayName() string {
	return a.Object.Properties.Name
}

func (a *Actor) Header() string {
	return a.Object.Properties.Image.URL
}

func (a *Actor) InboxURL() string {
	return a.Object.Properties.Inbox
}

func (a *Actor) Locked() bool {
	return a.Object.Properties.ManuallyApprovesFollowers
}

func (a *Actor) SharedInboxURL() string {
	return a.Object.Properties.Endpoints.SharedInbox
}

func (a *Actor) OutboxURL() string {
	return a.Object.Properties.Outbox
}

func (a *Actor) PublicKey() []byte {
	return []byte(a.Object.Properties.PublicKey.PublicKeyPem)
}

func (a *Actor) Note() string {
	return a.Object.Properties.Summary
}

func (a *Actor) URI() string {
	return a.Object.Properties.ID
}

func (a *Actor) AfterCreate(tx *gorm.DB) error {
	return forEach(tx, a.updateInstanceDomainsCount)
}

// func (a *Actor) AfterUpdate(tx *gorm.DB) error {
// 	return forEach(tx, a.maybeScheduleRefresh)
// }

func (a *Actor) AfterSave(tx *gorm.DB) error {
	peer := &Peer{
		Domain: a.Domain,
	}
	// save peer ignore if exists
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "domain"}},
		DoNothing: true,
	}).Create(peer).Error
}

func (a *Actor) updateInstanceDomainsCount(tx *gorm.DB) error {
	return tx.Model(&Instance{}).Where("1 = 1").UpdateColumns(map[string]interface{}{
		"domains_count": tx.Select("COUNT(distinct domain)").Model(&Actor{}),
	}).Error // update domain count on all instances.
}

// func (a *Actor) maybeScheduleRefresh(tx *gorm.DB) error {
// 	if !a.needsRefresh() {
// 		return nil
// 	}
// 	fmt.Println("scheduling refresh for", a.URI())
// 	return NewActors(tx).Refresh(a)
// }

// func (a *Actor) needsRefresh() bool {
// 	if a.OutboxURL() == "" || (a.InboxURL() == "" && a.SharedInboxURL() == "") {
// 		return true
// 	}
// 	return false
// }

func (a *Actor) Acct() string {
	if a.IsLocal() {
		return a.Name
	}
	return fmt.Sprintf("%s@%s", a.Name, a.Domain)
}

func (a *Actor) IsBot() bool {
	return !a.IsPerson()
}

func (a *Actor) IsPerson() bool {
	return a.Type == "Person" || a.Type == "LocalPerson"
}

// IsLocal indicates whether the actor is local to the instance.
func (a *Actor) IsLocal() bool {
	switch a.Type {
	case "LocalPerson", "LocalService":
		return true
	default:
		return false
	}
}

// IsRemote indicates whether the actor is not local to the instance.
func (a *Actor) IsRemote() bool {
	return !a.IsLocal()
}

func (a *Actor) IsGroup() bool {
	return a.Type == "Group"
}

func (a *Actor) ActorType() string {
	switch a.Type {
	case "LocalPerson":
		return "Person"
	case "LocalService":
		return "Service"
	default:
		return string(a.Type)
	}
}

func (a *Actor) PublicKeyID() string {
	return fmt.Sprintf("%s#main-key", a.URI())
}

func (a *Actor) URL() string {
	return fmt.Sprintf("https://%s/@%s", a.Domain, a.Name)
}

type Actors struct {
	db *gorm.DB
}

func NewActors(db *gorm.DB) *Actors {
	return &Actors{db: db}
}

// Find finds an account by its name and domain.
func (a *Actors) Find(name, domain string) (*Actor, error) {
	var actor Actor
	return &actor, a.db.Scopes(PreloadActor).Where("name = ? AND domain = ?", name, domain).Take(&actor).Error
}

// FindByURI returns an account by its URI if it exists locally.
func (a *Actors) FindByURI(uri string) (*Actor, error) {
	var actor []Actor
	if err := a.db.Scopes(PreloadActor).Joins("JOIN objects ON objects.id = actors.object_id").Where("objects.uri = ?", uri).Find(&actor).Error; err != nil {
		return nil, err
	}
	if len(actor) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &actor[0], nil
}

func (a *Actors) FindOrCreateByURI(uri string) (*Actor, error) {
	actor, err := a.FindByURI(uri)
	if err == nil {
		// found
		return actor, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// something went wrong
		return nil, err
	}
	// not found, create
	props, err := fetchObject(a.db.Statement.Context, uri)
	if err != nil {
		return nil, err
	}
	obj := &Object{
		Properties: props,
	}
	if err := a.db.Create(obj).Error; err != nil {
		return nil, err
	}
	return a.FindByURI(uri)
}

// Refesh schedules a refresh of an actor's data.
func (a *Actors) Refresh(actor *Actor) error {
	db := a.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "actor_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"created_at",
			"updated_at",
			"attempts", // resets the attempts counter
		}),
	})
	return db.Create(&ActorRefreshRequest{ActorID: actor.ObjectID}).Error
}

type Request struct {
	ID uint32 `gorm:"primarykey;"`
	// CreatedAt is the time the request was created.
	CreatedAt time.Time
	// UpdatedAt is the time the request was last updated.
	UpdatedAt time.Time
	// Attempts is the number of times the request has been attempted.
	Attempts uint32 `gorm:"not null;default:0"`
	// LastAttempt is the time the request was last attempted.
	LastAttempt time.Time
	// LastResult is the result of the last attempt if it failed.
	LastResult string `gorm:"type:text;"`
}

// ActorRefreshRequest is a request to refresh an actor's data.
type ActorRefreshRequest struct {
	Request
	// ActorID is the ID of the actor to refresh.
	ActorID snowflake.ID `gorm:"uniqueIndex;not null;"`
	// Actor is the actor to refresh.
	Actor *Actor `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
}

// MaybeExcludeReplies returns a query that excludes replies if the request contains
// the exclude_replies parameter.
func MaybeExcludeReplies(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if excludeReplies := parseBool(r, "exclude_replies"); excludeReplies {
			db = db.Where("in_reply_to_id IS NULL")
		}
		return db
	}
}

// MaybeExcludeReblogs returns a query that excludes reblogs if the request contains
// the exclude_reblogs parameter.
func MaybeExcludeReblogs(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if excludeReblogs := parseBool(r, "exclude_reblogs"); excludeReblogs {
			db = db.Where("reblog_id IS NULL")
		}
		return db
	}
}

// MaybePinned returns a query that only includes pinned statuses if the request contains
// the pinned parameter.
func MaybePinned(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if pinned := parseBool(r, "pinned"); pinned {
			db = db.Joins("JOIN reactions ON reactions.status_id = statuses.object_id AND reactions.pinned = true AND reactions.actor_id = statuses.actor_id")
		}
		return db
	}
}

// PreloadActor preloads all of an Actor's relations and associations.
func PreloadActor(query *gorm.DB) *gorm.DB {
	return query.Preload("Object")
}

// parseBool parses a boolean value from a request parameter.
// If the parameter is not present, it returns false.
// If the parameter is present but cannot be parsed, it returns false
func parseBool(r *http.Request, key string) bool {
	switch r.URL.Query().Get(key) {
	case "true", "1":
		return true
	default:
		return false
	}
}
