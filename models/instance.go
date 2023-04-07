package models

import (
	"time"

	"github.com/davecheney/pub/internal/snowflake"
)

// An Instance is an ActivityPub domain managed by this server.
// An Instance has many InstanceRules.
// An Instance has one Admin Account.
type Instance struct {
	snowflake.ID     `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt        time.Time
	Domain           string `gorm:"size:64;uniqueIndex"`
	AdminID          *snowflake.ID
	Admin            *Account `gorm:"<-:false;"`
	SourceURL        string
	Title            string `gorm:"size:64"`
	ShortDescription string
	Description      string
	Thumbnail        string         `gorm:"size:64"`
	AccountsCount    int            `gorm:"default:0;not null"`
	StatusesCount    int            `gorm:"default:0;not null"`
	DomainsCount     int32          `gorm:"default:0;not null"`
	Rules            []InstanceRule `gorm:"constraint:OnDelete:CASCADE;"`
}

type InstanceRule struct {
	ID         uint32 `gorm:"primarykey"`
	InstanceID uint64
	Text       string
}
