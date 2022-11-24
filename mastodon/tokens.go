package mastodon

import (
	"gorm.io/gorm"
)

type Token struct {
	gorm.Model
	UserID            uint
	User              User
	AccountID         uint
	Account           Account
	ApplicationID     uint
	AccessToken       string
	TokenType         string
	Scope             string
	AuthorizationCode string
}
