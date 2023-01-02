package models

import "github.com/davecheney/pub/internal/snowflake"

// An Application is a registered client application.
// An Application belongs to an Instance.
type Application struct {
	snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	InstanceID   snowflake.ID
	Instance     *Instance `gorm:"constraint:OnDelete:CASCADE;"`
	Name         string    `gorm:"size:64;not null"`
	Website      *string   `gorm:"size:64"`
	RedirectURI  string    `gorm:"size:128;not null"`
	ClientID     string    `gorm:"size:64;not null"`
	ClientSecret string    `gorm:"size:64;not null"`
	VapidKey     string    `gorm:"size:128;not null"`
}
