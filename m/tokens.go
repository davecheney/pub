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

type tokens struct {
	db *gorm.DB
}

func (t *tokens) FindByAccessToken(accessToken string) (*Token, error) {
	var token Token
	if err := t.db.Where("access_token = ?", accessToken).Joins("Account").First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}
