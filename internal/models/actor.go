package models

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/davecheney/m/internal/snowflake"
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
	Note           string `gorm:"not null"`
	FollowersCount int32  `gorm:"default:0;not null"`
	FollowingCount int32  `gorm:"default:0;not null"`
	StatusesCount  int32  `gorm:"default:0;not null"`
	LastStatusAt   time.Time
	Avatar         string `gorm:"size:255;not null"`
	Header         string `gorm:"size:255;not null"`
	PublicKey      []byte `gorm:"not null"`
	Attachments    []any  `gorm:"serializer:json"`
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

type Actors struct {
	db *gorm.DB
}

func NewActors(db *gorm.DB) *Actors {
	return &Actors{db: db}
}

// FindByURI returns an account by its URI if it exists locally.
func (a *Actors) FindByURI(uri string) (*Actor, error) {
	username, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	return a.Find(username, domain)
}

func (a *Actors) Find(name, domain string) (*Actor, error) {
	var actor Actor
	err := a.db.Where("name = ? AND domain = ?", name, domain).First(&actor).Error
	if err != nil {
		return nil, err
	}
	return &actor, nil
}

// FindOrCreate finds an account by its URI, or creates it if it doesn't exist.
func (a *Actors) FindOrCreate(uri string, createFn func(string) (*Actor, error)) (*Actor, error) {
	name, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	var actor Actor
	err = a.db.Where("name = ? AND domain = ?", name, domain).First(&actor).Error
	if err == nil {
		// found cached key
		return &actor, nil
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

func splitAcct(acct string) (string, string, error) {
	url, err := url.Parse(acct)
	if err != nil {
		return "", "", fmt.Errorf("splitAcct: %w", err)
	}
	return path.Base(url.Path), url.Host, nil
}
