package mastodon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/carlmjohnson/requests"
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

// FindOrCreateAccount finds an account by username and domain, or creates a new
// one if it doesn't exist.
func (a *Accounts) FindOrCreateAccount(uri string) (*Account, error) {
	username, domain, err := splitAcct(uri)
	var account Account
	err = a.db.Where("username = ? AND domain = ?", username, domain).First(&account).Error
	if err == nil {
		// found cached key
		return &account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var obj map[string]interface{}
	if err := requests.URL(uri).Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).ToJSON(&obj).Fetch(context.Background()); err != nil {
		return nil, err
	}

	account = Account{
		Username:       username,
		Domain:         domain,
		DisplayName:    obj["name"].(string),
		Locked:         obj["manuallyApprovesFollowers"].(bool),
		Bot:            obj["type"].(string) == "Service",
		Note:           obj["summary"].(string),
		URL:            obj["id"].(string),
		Avatar:         obj["icon"].(map[string]interface{})["url"].(string),
		AvatarStatic:   obj["icon"].(map[string]interface{})["url"].(string),
		Header:         obj["image"].(map[string]interface{})["url"].(string),
		HeaderStatic:   obj["image"].(map[string]interface{})["url"].(string),
		FollowersCount: int(obj["followers"].(map[string]interface{})["totalItems"].(float64)),
		FollowingCount: int(obj["following"].(map[string]interface{})["totalItems"].(float64)),
		StatusesCount:  int(obj["outbox"].(map[string]interface{})["totalItems"].(float64)),
		LastStatusAt:   time.Now(),
		PublicKey:      []byte(obj["publicKey"].(map[string]interface{})["publicKeyPem"].(string)),
	}
	if err := a.db.Create(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func splitAcct(acct string) (string, string, error) {
	url, err := url.Parse(acct)
	if err != nil {
		return "", "", fmt.Errorf("splitAcct: %w", err)
	}
	return path.Base(url.Path), url.Host, nil
}
