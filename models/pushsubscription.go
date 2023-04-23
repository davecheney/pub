package models

import (
	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type PushSubscription struct {
	ID            uint32 `gorm:"primaryKey"`
	AccountID     snowflake.ID
	Account       *Account `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
	Endpoint      string   `gorm:"not null"`
	Mention       bool
	Status        bool
	Reblog        bool
	Follow        bool
	FollowRequest bool
	Favourite     bool
	Poll          bool
	Update        bool
	Policy        PushSubscriptionPolicy `gorm:"not null;default:'all'"`
}

type PushSubscriptionPolicy string

func (PushSubscriptionPolicy) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "postgres":
		return "enum('all', 'followed', 'follower', 'none')"
	case "sqlite":
		return "TEXT"
	default:
		return ""
	}
}
