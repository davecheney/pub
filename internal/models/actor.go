package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
)

type Actor struct {
	snowflake.ID   `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt      time.Time
	Type           string `gorm:"type:enum('Person', 'Application', 'Service', 'Group', 'Organization', 'LocalPerson');default:'Person';not null"`
	URI            string `gorm:"uniqueIndex;size:128;not null"`
	Name           string `gorm:"size:64;uniqueIndex:idx_actor_name_domain;not null"`
	Domain         string `gorm:"size:64;uniqueIndex:idx_actor_name_domain;not null"`
	DisplayName    string `gorm:"size:128;not null"`
	Locked         bool   `gorm:"default:false;not null"`
	Note           string `gorm:"type:text"` // max 2^16
	FollowersCount int32  `gorm:"default:0;not null"`
	FollowingCount int32  `gorm:"default:0;not null"`
	StatusesCount  int32  `gorm:"default:0;not null"`
	LastStatusAt   time.Time
	Avatar         string            `gorm:"size:255"`
	Header         string            `gorm:"size:255"`
	PublicKey      []byte            `gorm:"type:blob;not null"`
	Attributes     []*ActorAttribute `gorm:"constraint:OnDelete:CASCADE;"`
}

func (a *Actor) AfterCreate(tx *gorm.DB) error {
	return forEach(tx, a.updateInstanceDomainsCount)
}

func (a *Actor) updateInstanceDomainsCount(tx *gorm.DB) error {
	return tx.Model(&Instance{}).Where("1 = 1").UpdateColumns(map[string]interface{}{
		"domains_count": tx.Select("COUNT(distinct domain)").Model(&Actor{}),
	}).Error // update domain count on all instances.
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

func (a *Actor) IsLocal() bool {
	return a.Type == "LocalPerson"
}

func (a *Actor) IsGroup() bool {
	return a.Type == "Group"
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
	if err := a.db.Where(&Actor{URI: uri}).Find(&actors).Error; err != nil {
		return nil, err
	}
	if len(actors) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &actors[0], nil
}

// PreloadActor preloads all of an Actor's relations and associations.
func PreloadActor(query *gorm.DB) *gorm.DB {
	return query.Preload("Attributes")
}
