package mastodon

import (
	"gorm.io/gorm"
)

type Token struct {
	gorm.Model
	UserID            uint
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
	result := t.db.Where("access_token = ?", accessToken).First(token)
	return token, result.Error
}

func (t *tokens) findByAuthorizationCode(code string) (*Token, error) {
	token := &Token{}
	result := t.db.Where("authorization_code = ?", code).First(token)
	return token, result.Error
}

func (t *tokens) create(token *Token) error {
	result := t.db.Create(token)
	return result.Error
}
