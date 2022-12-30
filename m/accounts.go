package m

import (
	"fmt"
	"net/url"
	"path"

	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

type Marker struct {
	gorm.Model
	AccountID  uint32
	Name       string `gorm:"size:32"`
	Version    int    `gorm:"default:0"`
	LastReadId uint
}

type accounts struct {
	db      *gorm.DB
	service *Service
}

// FindByURI returns an account by its URI if it exists locally.
func (a *accounts) FindByURI(uri string) (*models.Account, error) {
	username, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	var account models.Account
	if err := a.db.Where("username = ? AND domain = ?", username, domain).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (a *accounts) Find(id uint64) (*models.Account, error) {
	var account models.Account
	if err := a.db.Where("actor_id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (a *accounts) FindAdminAccount() (*models.Account, error) {
	i, err := a.service.Instances().FindByDomain("cheney.net")
	if err != nil {
		return nil, err
	}
	return i.Admin, nil
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
