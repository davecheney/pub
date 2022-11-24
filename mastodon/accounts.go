package mastodon

import (
	"time"

	"gorm.io/gorm"
)

type Account struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// User *User `gorm:"foreignKey:ID;references:user_id"`

	Username       string
	Domain         string
	Acct           string
	DisplayName    string
	Locked         bool
	Bot            bool
	Note           string
	URL            string
	Avatar         string
	AvatarStatic   string
	Header         string
	HeaderStatic   string
	FollowersCount int
	FollowingCount int
	StatusesCount  int
	LastStatusAt   time.Time
}

type accounts struct {
	db *gorm.DB
}

func (a *accounts) findByID(id int) (*Account, error) {
	account := &Account{}
	result := a.db.First(account, id)
	return account, result.Error
}

func (a *accounts) findByUserID(id int) (*Account, error) {
	return nil, nil
	// account := &Account{}
	// err := a.db.QueryRowx(`SELECT * FROM mastodon_accounts WHERE user_id = ?`, id).StructScan(account)
	// return account.hydrate(), err
}

func (a *accounts) findByAcct(acct string) (*Account, error) {
	return nil, nil
	// parts := strings.Split(strings.TrimPrefix(acct, "acct:"), "@")
	// if len(parts) != 2 {
	// 	return nil, fmt.Errorf("invalid acct: %s", acct)
	// }
	// account := &Account{}
	// err := a.db.QueryRowx(`SELECT * FROM mastodon_accounts WHERE username = ? AND domain = ?`, parts[0], parts[1]).StructScan(account)
	// return account.hydrate(), err
}
