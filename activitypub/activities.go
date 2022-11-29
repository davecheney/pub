package activitypub

import (
	"time"

	"github.com/davecheney/m/m"
	"gorm.io/gorm"
)

type Activity struct {
	gorm.Model
	AccountID    uint
	Account      *m.Account
	Activity     map[string]interface{} `gorm:"serializer:json"`
	ActivityType string
	ObjectType   string
	ProcessedAt  *time.Time
}

func (Activity) TableName() string {
	return "activitypub_inbox"
}
