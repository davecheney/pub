package activitypub

import (
	"time"

	"github.com/davecheney/m/mastodon"
	"gorm.io/gorm"
)

type Activity struct {
	gorm.Model
	AccountID    uint
	Account      *mastodon.Account
	Activity     map[string]interface{} `gorm:"serializer:json"`
	ActivityType string
	ObjectType   string
	ProcessedAt  *time.Time
}

func (Activity) TableName() string {
	return "activitypub_inbox"
}
