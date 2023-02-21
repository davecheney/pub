package models

import (
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// A Token is an access token for an Application.
// A Token belongs to an Account.
// A Token belongs to an Application.
type Token struct {
	AccessToken       string `gorm:"size:64;primaryKey;autoIncrement:false"`
	CreatedAt         time.Time
	AccountID         snowflake.ID
	Account           *Account `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
	ApplicationID     snowflake.ID
	Application       *Application `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
	TokenType         `gorm:"not null"`
	Scope             string `gorm:"size:64;not null"`
	AuthorizationCode string `gorm:"size:64;not null"`
}

type TokenType string

func (TokenType) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "postgres":
		return "enum('Bearer')"
	case "sqlite":
		return "TEXT"
	default:
		return ""
	}
}
