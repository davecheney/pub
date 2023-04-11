package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/webfinger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type Actor struct {
	snowflake.ID   `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt      time.Time
	Type           ActorType `gorm:"default:'Person';not null"`
	URI            string    `gorm:"uniqueIndex;size:128;not null"`
	Name           string    `gorm:"size:64;uniqueIndex:idx_actor_name_domain;not null"`
	Domain         string    `gorm:"size:64;uniqueIndex:idx_actor_name_domain;not null"`
	DisplayName    string    `gorm:"size:128;not null"`
	Locked         bool      `gorm:"default:false;not null"`
	Note           string    `gorm:"type:text"` // max 2^16
	FollowersCount int32     `gorm:"default:0;not null"`
	FollowingCount int32     `gorm:"default:0;not null"`
	StatusesCount  int32     `gorm:"default:0;not null"`
	LastStatusAt   time.Time
	Avatar         string            `gorm:"size:255"`
	Header         string            `gorm:"size:255"`
	PublicKey      []byte            `gorm:"size:16384;type:blob;not null"`
	Attributes     []*ActorAttribute `gorm:"constraint:OnDelete:CASCADE;"`
	InboxURL       string            `gorm:"size:255;not null;default:''"`
	OutboxURL      string            `gorm:"size:255;not null;default:''"`
	SharedInboxURL string            `gorm:"size:255;not null;default:''"`
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
	if a.SharedInboxURL != "" {
		return a.SharedInboxURL
	}
	return a.InboxURL
}

func (a *Actor) AfterCreate(tx *gorm.DB) error {
	return forEach(tx, a.updateInstanceDomainsCount)
}

func (a *Actor) AfterUpdate(tx *gorm.DB) error {
	return forEach(tx, a.maybeScheduleRefresh)
}

func (a *Actor) updateInstanceDomainsCount(tx *gorm.DB) error {
	return tx.Model(&Instance{}).Where("1 = 1").UpdateColumns(map[string]interface{}{
		"domains_count": tx.Select("COUNT(distinct domain)").Model(&Actor{}),
	}).Error // update domain count on all instances.
}

func (a *Actor) maybeScheduleRefresh(tx *gorm.DB) error {
	if !a.needsRefresh() {
		return nil
	}
	acct := webfinger.Acct{User: a.Name, Host: a.Domain}
	fmt.Println("scheduling refresh for", acct.String())
	return NewActors(tx).Refresh(a)
}

func (a *Actor) needsRefresh() bool {
	if a.OutboxURL == "" || (a.InboxURL == "" && a.SharedInboxURL == "") {
		return true
	}
	return false
}

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
	return fmt.Sprintf("%s#main-key", a.URI)
}

func (a *Actor) URL() string {
	return fmt.Sprintf("https://%s/@%s", a.Domain, a.Name)
}

type ActorAttribute struct {
	ID      uint32 `gorm:"primarykey"`
	ActorID snowflake.ID
	Name    string `gorm:"size:255;not null"`
	Value   string `gorm:"type:text;not null"`
}

type Actors struct {
	db *gorm.DB
}

func NewActors(db *gorm.DB) *Actors {
	return &Actors{db: db}
}

// FindOrCreate finds an account by its URI, or creates it if it doesn't exist.
func (a *Actors) FindOrCreate(uri string, createFn func(string) (*Actor, error)) (*Actor, error) {
	actor, err := a.FindByURI(uri)
	if err == nil {
		// found cached key
		return actor, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	acc, err := createFn(uri)
	if err != nil {
		return nil, err
	}
	err = a.db.Create(acc).Error
	return acc, err
}

// FindByURI returns an account by its URI if it exists locally.
func (a *Actors) FindByURI(uri string) (*Actor, error) {
	// use find to avoid record not found error in case of empty result
	var actors []Actor
	if err := a.db.Limit(1).Find(&actors, "uri = ?", uri).Error; err != nil {
		return nil, err
	}
	if len(actors) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &actors[0], nil
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
	return db.Create(&ActorRefreshRequest{ActorID: actor.ID}).Error
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
			db = db.Joins("JOIN reactions ON reactions.status_id = statuses.id AND reactions.pinned = true AND reactions.actor_id = statuses.actor_id")
		}
		return db
	}
}

// PreloadActor preloads all of an Actor's relations and associations.
func PreloadActor(query *gorm.DB) *gorm.DB {
	return query.Preload("Attributes")
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
