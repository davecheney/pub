package mastodon

import (
	"fmt"
	"strings"
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

func (a *Account) hydrate() *Account {
	if a == nil {
		return a
	}
	a.URI = fmt.Sprintf("https://%s/users/%s", a.Domain, a.Username)
	a.URL = fmt.Sprintf("https://%s/@%s", a.Domain, a.Username)
	a.Acct = a.Username
	return a
}

func (a *accounts) findByID(id int) (*Account, error) {
	account := &Account{}
	err := a.db.QueryRowx(`SELECT * FROM mastodon_accounts WHERE id = ?`, id).StructScan(account)
	return account.hydrate(), err
}

func (a *accounts) findByUserID(id int) (*Account, error) {
	account := &Account{}
	err := a.db.QueryRowx(`SELECT * FROM mastodon_accounts WHERE user_id = ?`, id).StructScan(account)
	return account.hydrate(), err
}

func (a *accounts) findByAcct(acct string) (*Account, error) {
	parts := strings.Split(strings.TrimPrefix(acct, "acct:"), "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid acct: %s", acct)
	}
	account := &Account{}
	err := a.db.QueryRowx(`SELECT * FROM mastodon_accounts WHERE username = ? AND domain = ?`, parts[0], parts[1]).StructScan(account)
	return account.hydrate(), err
}
