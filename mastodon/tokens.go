package mastodon

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type tokens struct {
	db *gorm.DB
}

func (t *tokens) findByAuthorizationCode(code string) (*Token, error) {
	token := &Token{}
	result := t.db.Preload(clause.Associations).Where("authorization_code = ?", code).First(token)
	return token, result.Error
}
