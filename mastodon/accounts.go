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
	InstanceID     uint
	Instance       *Instance
	Domain         string `gorm:"uniqueIndex:idx_domainusername;size:64"`
	Username       string `gorm:"uniqueIndex:idx_domainusername;size:64"`
	DisplayName    string `gorm:"size:64"`
	Email          string `gorm:"size:64"`
	Local          bool
	Locked         bool
	Bot            bool
	Note           string
	Avatar         string
	AvatarStatic   string
	Header         string
	HeaderStatic   string
	FollowersCount int `gorm:"default:0;not null"`
	FollowingCount int `gorm:"default:0;not null"`
	StatusesCount  int `gorm:"default:0;not null"`
	LastStatusAt   time.Time

	EncryptedPassword []byte // only used for local accounts
	PublicKey         []byte
	PrivateKey        []byte // only used for local accounts

	Statuses []Status
}

func (a *Account) AfterCreate(tx *gorm.DB) error {
	// update count of accounts on instance
	var instance Instance
	if err := tx.Where("domain = ?", a.Domain).First(&instance).Error; err != nil {
		return err
	}
	var count int64
	if err := tx.Model(&Account{}).Where("domain = ?", a.Domain).Count(&count).Error; err != nil {
		return err
	}
	instance.AccountsCount = int(count)
	return tx.Save(&instance).Error
}

func (a *Account) Acct() string {
	if a.Local {
		return a.Username
	}
	return a.Username + "@" + a.Domain
}

func (a *Account) PublicKeyID() string {
	return fmt.Sprintf("https://%s/users/%s#main-key", a.Domain, a.Username)
}

func (a *Account) serialize() map[string]any {
	return map[string]any{
		"id":       strconv.Itoa(int(a.ID)),
		"username": a.Username,

		"acct":            a.Acct(),
		"display_name":    a.DisplayName,
		"locked":          a.Locked,
		"bot":             a.Bot,
		"created_at":      a.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
		"note":            a.Note,
		"url":             fmt.Sprintf("https://%s/@%s", a.Domain, a.Username),
		"avatar":          stringOrDefault(a.Avatar, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"avatar_static":   stringOrDefault(a.AvatarStatic, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"header":          stringOrDefault(a.Header, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"header_static":   stringOrDefault(a.HeaderStatic, fmt.Sprintf("https://%s/header.png", a.Domain)),
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

func (a *Accounts) instances() *Instances {
	return NewInstances(a.db, "")
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

func (a *Accounts) Relationships(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	var token Token
	if err := a.db.Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var account Account
	id := r.URL.Query().Get("id")
	if err := a.db.Where("id = ?", id).First(&account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// todo
}

// FindOrCreateAccount finds an account by username and domain, or creates a new
// one if it doesn't exist.
func (a *Accounts) FindOrCreateAccount(uri string) (*Account, error) {
	username, domain, err := splitAcct(uri)
	instance, err := a.instances().FindOrCreateInstance(domain)
	if err != nil {
		return nil, err
	}

	var account Account
	err = a.db.Where("username = ? AND domain = ?", username, instance.Domain).First(&account).Error
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
		InstanceID:     instance.ID,
		Instance:       instance,
		DisplayName:    stringFromAny(obj["name"]),
		Locked:         boolFromAny(obj["manuallyApprovesFollowers"]),
		Bot:            stringFromAny(obj["type"]) == "Service",
		Note:           stringFromAny(obj["summary"]),
		Avatar:         stringFromAny(mapFromAny(obj["icon"])["url"]),
		AvatarStatic:   stringFromAny(mapFromAny(obj["icon"])["url"]),
		Header:         stringFromAny(mapFromAny(obj["image"])["url"]),
		HeaderStatic:   stringFromAny(mapFromAny(obj["image"])["url"]),
		FollowersCount: 0,
		FollowingCount: 0,
		StatusesCount:  0,
		LastStatusAt:   time.Now(),

		PublicKey: []byte(stringFromAny(mapFromAny(obj["publicKey"])["publicKeyPem"])),
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

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}
