package mastodon

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Account struct {
	ID             int       `json:"id" db:"id"`
	UserID         int       `json:"-" db:"user_id"`
	Username       string    `json:"username" db:"username"`
	Domain         string    `json:"-" db:"domain"`
	DisplayName    string    `json:"display_name" db:"display_name"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"-" db:"updated_at"`
	PublicKey      string    `json:"-" db:"public_key"`
	PrivateKey     string    `json:"-" db:"private_key"`
	Note           string    `json:"note" db:"note"`
	FollowersCount int       `json:"followers_count" db:"followers_count"`
	FollowingCount int       `json:"following_count" db:"following_count"`
	StatusesCount  int       `json:"statuses_count" db:"statuses_count"`

	// synthesised by findAccount* functions
	URI  string `json:"uri" db:"-"`
	URL  string `json:"url" db:"-"`
	Acct string `json:"acct" db:"-"`
}

type accounts struct {
	db *sqlx.DB
}

func (a *accounts) findByUserID(id int) (*Account, error) {
	account := &Account{}
	err := a.db.QueryRowx(`SELECT * FROM mastodon_accounts WHERE user_id = ?`, id).StructScan(account)
	if err != nil {
		return nil, err
	}
	account.URI = fmt.Sprintf("https://%s/users/%s", account.Domain, account.Username)
	account.URL = fmt.Sprintf("https://%s/@%s", account.Domain, account.Username)
	account.Acct = account.Username
	return account, nil
}
