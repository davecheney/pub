package mastodon

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
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

func (a *Account) serialize() map[string]any {
	return map[string]any{
		"id":              strconv.Itoa(int(a.ID)),
		"username":        a.Username,
		"acct":            a.Acct,
		"display_name":    a.DisplayName,
		"locked":          a.Locked,
		"bot":             a.Bot,
		"created_at":      a.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
		"note":            a.Note,
		"url":             a.URL,
		"avatar":          a.Avatar,
		"avatar_static":   a.Avatar,
		"header":          a.Header,
		"header_static":   a.Header,
		"followers_count": a.FollowersCount,
		"following_count": a.FollowingCount,
		"statuses_count":  a.StatusesCount,
		"last_status_at":  a.LastStatusAt.Format("2006-01-02T15:04:05.006Z"),
		"emojis":          []map[string]any{},
		"fields":          []map[string]any{},
	}
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
