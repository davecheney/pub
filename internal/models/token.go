package models

import (
	"time"
)

// A Token is an access token for an Application.
// A Token belongs to an Account.
// A Token belongs to an Application.
type Token struct {
	AccessToken       string `gorm:"size:64;primaryKey;autoIncrement:false"`
	CreatedAt         time.Time
	AccountID         uint64
	Account           *Account
	ApplicationID     uint64
	Application       *Application
	TokenType         string `gorm:"type:enum('Bearer');not null"`
	Scope             string `gorm:"size:64;not null"`
	AuthorizationCode string `gorm:"size:64;not null"`
}
