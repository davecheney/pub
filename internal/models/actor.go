package models

import (
	"fmt"
	"time"

	"github.com/davecheney/m/internal/snowflake"
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
