package mastodon

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type Token struct {
	ID                int       `json:"-" db:"id"`
	UserID            int       `json:"-" db:"user_id"`
	ApplicationID     int       `json:"-" db:"application_id"`
	CreatedAt         time.Time `json:"-" db:"created_at"`
	AccessToken       string    `json:"access_token" db:"access_token"`
	TokenType         string    `json:"token_type" db:"token_type"`
	Scope             string    `json:"scope" db:"scope"`
	AuthorizationCode string    `json:"-" db:"authorization_code"`
}

type tokens struct {
	db *sqlx.DB
}

func (t *tokens) findByAccessToken(accessToken string) (*Token, error) {
	token := &Token{}
	err := t.db.QueryRowx(`SELECT * FROM tokens WHERE access_token = ?`, accessToken).StructScan(token)
	return token, err
}

func (t *tokens) findByAuthorizationCode(code string) (*Token, error) {
	token := &Token{}
	err := t.db.QueryRowx(`SELECT * FROM tokens WHERE authorization_code = ?`, code).StructScan(token)
	return token, err
}
