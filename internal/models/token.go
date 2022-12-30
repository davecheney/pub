package models

import (
	"time"

	"github.com/davecheney/m/internal/snowflake"
)

// A Token is an access token for an Application.
// A Token belongs to an Account.
// A Token belongs to an Application.
type Token struct {
	AccessToken       string `gorm:"size:64;primaryKey;autoIncrement:false"`
	CreatedAt         time.Time
	AccountID         snowflake.ID
	Account           *Account
	ApplicationID     snowflake.ID
	Application       *Application
	TokenType         string `gorm:"type:enum('Bearer');not null"`
	Scope             string `gorm:"size:64;not null"`
	AuthorizationCode string `gorm:"size:64;not null"`
}
