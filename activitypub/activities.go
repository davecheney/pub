package activitypub

import (
	"gorm.io/gorm"
)

type Activity struct {
	gorm.Model
	AccountID    uint
	Activity     map[string]interface{} `gorm:"serializer:json"`
	ActivityType string
	ObjectType   string
}

func (Activity) TableName() string {
	return "activitypub_inbox"
}
