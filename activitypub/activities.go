package activitypub

import (
	"time"

	"gorm.io/gorm"
)

type Activity struct {
	gorm.Model
	Activity     string
	ActivityType string
	ObjectType   string
	ProcessedAt  *time.Time
}

func (Activity) TableName() string {
	return "activitypub_inbox"
}
