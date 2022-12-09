package activitypub

import "gorm.io/gorm"

type Actor struct {
	gorm.Model
	Type string `gorm:"type:enum('Person', 'Application', 'Service', 'Group', 'Organization')"`
}
