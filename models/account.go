package models

import (
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
)

// An Account is a user account on an Instance.
// An Account belongs to an Actor.
// An Account belongs to an Instance.
type Account struct {
	snowflake.ID      `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt         time.Time
	InstanceID        snowflake.ID
	Instance          *Instance `gorm:"<-:false;"`
	ActorID           snowflake.ID
	Actor             *Actor          `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
	Lists             []AccountList   `gorm:"constraint:OnDelete:CASCADE;"`
	Markers           []AccountMarker `gorm:"constraint:OnDelete:CASCADE;"`
	Email             string          `gorm:"size:64;not null"`
	EncryptedPassword []byte          `gorm:"size:60;not null"`
	PrivateKey        []byte          `gorm:"not null"`
	RoleID            uint32
	Role              *AccountRole
}

func (a *Account) Name() string {
	return a.Actor.Name
}

func (a *Account) Domain() string {
	return a.Actor.Domain
}

type AccountRole struct {
	ID          uint32 `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string `gorm:"size:16;not null"`
	Color       string `gorm:"size:8;not null,default:''"`
	Position    int32
	Permissions uint32
	Highlighted bool
}

type AccountList struct {
	snowflake.ID  `gorm:"primarykey;autoIncrement:false"`
	AccountID     snowflake.ID        `gorm:"not null;"`
	Title         string              `gorm:"size:64"`
	RepliesPolicy string              `gorm:"enum('public','followers','none');not null;default:'public'"`
	Members       []AccountListMember `gorm:"constraint:OnDelete:CASCADE;"`
}

type AccountListMember struct {
	AccountListID snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	MemberID      snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Member        *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
}

type AccountMarker struct {
	ID         uint32 `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	AccountID  snowflake.ID `gorm:"not null;uniqueIndex:idx_account_name;index"`
	Name       string       `gorm:"enum('home','notifications');not null;uniqueIndex:idx_account_name"`
	Version    int32        `gorm:"not null;"`
	LastReadID snowflake.ID `gorm:"not null;"`
}

type Accounts struct {
	db *gorm.DB
}

func NewAccounts(db *gorm.DB) *Accounts {
	return &Accounts{db: db}
}

func (a *Accounts) AccountForActor(actor *Actor) (*Account, error) {
	var account Account
	if err := a.db.Joins("Actor").First(&account, "actor_id = ?", actor.ID).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

type AccountPreferences struct {
	AccountID                snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	PostingDefaultVisibility string       `gorm:"enum('public', 'unlisted', 'private', 'direct');not null;default:'public'"`
	PostingDefaultSensitive  bool         `gorm:"not null;default:false"`
	PostingDefaultLanguage   string       `gorm:"size:8;"`
	ReadingExpandMedia       string       `gorm:"enum('default','show_all','hide_all');not null;default:'default'"`
	ReadingExpandSpoilers    bool         `gorm:"not null;default:false"`
}
