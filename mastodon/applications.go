package mastodon

import (
	"gorm.io/gorm"
)

type Application struct {
	gorm.Model
	Name         string
	Website      *string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	VapidKey     string
}
