// Package mastodon implements a Mastodon API service.
package mastodon

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"gorm.io/gorm"
)

type Env struct {
	*models.Env
}

// authenticate authenticates the bearer token attached to the request and, if
// successful, returns the account associated with the token.
func (e *Env) authenticate(r *http.Request) (*models.Account, error) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token models.Token
	if err := e.DB.Joins("Account").Preload("Account.Actor").Preload("Account.Role").Take(&token, "access_token = ?", bearer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Error(http.StatusUnauthorized, err)
		}
		return nil, err
	}
	return token.Account, nil
}

func (e *Env) findByDomain(domain string) (*models.Instance, error) {
	var instance models.Instance
	if err := e.DB.Where("domain = ?", domain).Preload("Admin").Preload("Admin.Actor").Preload("Rules").Take(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Error(http.StatusNotFound, err)
		}
		return nil, err
	}
	return &instance, nil
}

func stringOrDefault(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

func stringOrNull[T number](v *T) any {
	if v == nil {
		return nil
	}
	return strconv.Itoa(int(*v))
}

type number interface {
	~uint | ~uint64 | ~uint32
}

func toString[T number](n T) string {
	return strconv.FormatUint(uint64(n), 10)
}

func utoa(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}
