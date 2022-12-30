package models

import (
	"time"

	"github.com/davecheney/m/internal/snowflake"
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
	Actor             *Actor
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
