package models

import "github.com/davecheney/pub/internal/snowflake"

// An Application is a registered client application.
// An Application belongs to an Instance.
type Application struct {
	snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	InstanceID   snowflake.ID
	Instance     *Instance `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
	Name         string    `gorm:"size:255;not null"`
	Website      string    `gorm:"size:255"`
	RedirectURI  string    `gorm:"size:255;not null"`
	ClientID     string    `gorm:"size:255;not null"`
	ClientSecret string    `gorm:"size:255;not null"`
	VapidKey     string    `gorm:"size:255;not null"`
	Scopes       string    `gorm:"size:255;not null;default:''"`
}
