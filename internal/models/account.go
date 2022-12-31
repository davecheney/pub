package models

import (
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"gorm.io/gorm"
)

// An Account is a user account on an Instance.
// An Account belongs to an Actor.
// An Account belongs to an Instance.
type Account struct {
	snowflake.ID      `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt         time.Time
	InstanceID        snowflake.ID
	Instance          *Instance
	ActorID           snowflake.ID
	Actor             *Actor `gorm:"constraint:OnDelete:CASCADE;"`
	Lists             []AccountList
	Email             string `gorm:"size:64;not null"`
	EncryptedPassword []byte `gorm:"size:60;not null"`
	PrivateKey        []byte `gorm:"not null"`
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
	ID            uint32 `gorm:"primarykey"`
	CreatedAt     time.Time
	AccountID     uint64
	Title         string `gorm:"size:64"`
	RepliesPolicy string `gorm:"size:64"`
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
