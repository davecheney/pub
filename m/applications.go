package m

import (
	"gorm.io/gorm"
)

type Application struct {
	gorm.Model
	InstanceID   uint
	Instance     *Instance
	Name         string
	Website      *string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	VapidKey     string
}
