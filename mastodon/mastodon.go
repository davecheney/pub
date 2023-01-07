// Package mastodon implements a Mastodon API service.
package mastodon

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
)

type Env struct {
	*models.Env
}

// authenticate authenticates the bearer token attached to the request and, if
// successful, returns the account associated with the token.
func (e *Env) authenticate(r *http.Request) (*models.Account, error) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if bearer == "" {
		return nil, httpx.Error(http.StatusUnauthorized, errors.New("missing bearer token"))
	}
	var token models.Token
	if err := e.DB.Joins("Account").Preload("Account.Actor").Preload("Account.Role").Take(&token, "access_token = ?", bearer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Error(http.StatusUnauthorized, err)
		}
		return nil, err
	}
	return token.Account, nil
}

func linkHeader(w http.ResponseWriter, r *http.Request, newest, oldest snowflake.ID) {
	w.Header().Add("Link", fmt.Sprintf(`<https://%s%s?max_id=%d>; rel="next"`, r.Host, r.URL.Path, oldest))
	w.Header().Add("Link", fmt.Sprintf(`<https://%s%s?min_id=%d>; rel="prev"`, r.Host, r.URL.Path, newest))
}

func stringOrDefault(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}
