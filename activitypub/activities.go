package activitypub

import (
	"time"

	"gorm.io/gorm"
)

type Activity struct {
	gorm.Model
	Activity     map[string]interface{} `gorm:"serializer:json"`
	ActivityType string
	ObjectType   string
	ProcessedAt  *time.Time
}

func (Activity) TableName() string {
	return "activitypub_inbox"
}
