package mastodon

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Token struct {
	gorm.Model
	User              User    `gorm:"foreignKey:ID"`
	Account           Account `gorm:"foreignKey:ID"`
	ApplicationID     uint
	AccessToken       string `json:"access_token"`
	TokenType         string `json:"token_type"`
	Scope             string `json:"scope"`
	AuthorizationCode string `json:"-"`
}

type tokens struct {
	db *gorm.DB
}

func (t *tokens) findByAccessToken(accessToken string) (*Token, error) {
	token := &Token{}
	result := t.db.Preload(clause.Associations).Where("access_token = ?", accessToken).First(token)
	return token, result.Error
}

func (t *tokens) findByAuthorizationCode(code string) (*Token, error) {
	token := &Token{}
	result := t.db.Preload(clause.Associations).Where("authorization_code = ?", code).First(token)
	return token, result.Error
}
