// Package mastodon implements a Mastodon API service.
package mastodon

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/streaming"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

type Env struct {
	*gorm.DB
	*streaming.Mux
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

// sortStatuses sorts the statuses by their ID, in descending order.
func sortStatuses(statuses []*models.Status) {
	sort.SliceStable(statuses, func(i, j int) bool {
		return statuses[i].ID > statuses[j].ID
	})
}
