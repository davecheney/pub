package mastodon

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	Username       string `gorm:"uniqueIndex:idx_usernamedomain"`
	Domain         string `gorm:"uniqueIndex:idx_usernamedomain"`
	DisplayName    string
	Locked         bool
	Bot            bool
	Note           string
	URL            string `gorm:"uniqueIndex:idx_url"`
	Avatar         string
	AvatarStatic   string
	Header         string
	HeaderStatic   string
	FollowersCount int
	FollowingCount int
	StatusesCount  int
	LastStatusAt   time.Time

	EncryptedPassword []byte // only used for local accounts
	PublicKey         []byte
	PrivateKey        []byte // only used for local accounts

	Statuses []Status
}

func (a *Account) Acct() string {
	return a.Username + "@" + a.Domain
}

func (a *Account) serialize() map[string]any {
	return map[string]any{
		"id":              strconv.Itoa(int(a.ID)),
		"username":        a.Username,
		"acct":            a.Acct(),
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

type Accounts struct {
	db *gorm.DB
}

func NewAccounts(db *gorm.DB) *Accounts {
	return &Accounts{db: db}
}

func (a *Accounts) VerifyCredentials(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	var token Token
	if err := a.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, token.Account.serialize())
}
