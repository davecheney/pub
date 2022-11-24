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

	UserID uint

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

func (a *accounts) findByID(id uint) (*Account, error) {
	account := &Account{}
	result := a.db.First(account, id)
	return account, result.Error
}

func (a *accounts) findByUserID(id uint) (*Account, error) {
	account := &Account{}
	result := a.db.Where("user_id = ?", id).First(account)
	return account, result.Error
}

func (a *accounts) findByAcct(acct string) (*Account, error) {
	account := &Account{}
	result := a.db.Where("acct = ?", acct[5:]).First(account)
	return account, result.Error
}
