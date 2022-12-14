package m

import (
	"fmt"
	"net/url"
	"path"

	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	InstanceID        uint
	Instance          *Instance
	ActorID           uint64
	Actor             *Actor
	Notifications     []Notification
	Markers           []Marker
	Lists             []AccountList
	Email             string
	EncryptedPassword []byte
	PrivateKey        []byte `gorm:"not null"`
}

type Marker struct {
	gorm.Model
	AccountID  uint
	Name       string `gorm:"size:32"`
	Version    int    `gorm:"default:0"`
	LastReadId uint
}

type Notification struct {
	gorm.Model
	AccountID uint
	Account   *Account
	StatusID  *uint
	Status    *Status
	Type      string `gorm:"size:64"`
}

func (a *Account) Name() string {
	return a.Actor.Name
}

func (a *Account) Domain() string {
	return a.Actor.Domain
}

func (a *Account) Acct() string {
	if a.isLocal() {
		return a.Name()
	}
	return a.Name() + "@" + a.Domain()
}

func (a *Account) isLocal() bool {
	return a.Actor.Type == "LocalPerson"
}

type accounts struct {
	db      *gorm.DB
	service *Service
}

// FindByURI returns an account by its URI if it exists locally.
func (a *accounts) FindByURI(uri string) (*Account, error) {
	username, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	var account Account
	if err := a.db.Where("username = ? AND domain = ?", username, domain).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (a *accounts) Find(id uint64) (*Account, error) {
	var account Account
	if err := a.db.Where("actor_id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (a *accounts) FindAdminAccount() (*Account, error) {
	var account Account
	if err := a.db.Where("Actor.name = ? AND Actor.domain = ?", "dave", "cheney.net").Joins("Actor").First(&account).Error; err != nil {
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

type AccountList struct {
	gorm.Model
	AccountID     uint
	Title         string `gorm:"size:64"`
	RepliesPolicy string `gorm:"size:64"`
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
