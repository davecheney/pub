package m

import (
	"gorm.io/gorm"
)

type Token struct {
	gorm.Model
	AccountID         uint
	Account           *Account
	ApplicationID     uint
	AccessToken       string
	TokenType         string
	Scope             string
	AuthorizationCode string
}
