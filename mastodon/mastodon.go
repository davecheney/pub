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
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type Env struct {
	*gorm.DB
	*streaming.Mux
	Logger *slog.Logger
}

func (e *Env) Log() *slog.Logger {
	return e.Logger
}

// authenticate authenticates the bearer token attached to the request and, if
// successful, returns the account associated with the token.
func (e *Env) authenticate(r *http.Request) (*models.Account, error) {
	bearer := r.Header.Get("Authorization")
	if bearer == "" {
		return nil, httpx.Error(http.StatusUnauthorized, errors.New("missing bearer token"))
	}
	if !strings.HasPrefix(bearer, "Bearer ") {
		return nil, httpx.Error(http.StatusUnauthorized, errors.New("invalid bearer token"))
	}
	bearer = strings.TrimPrefix(bearer, "Bearer ")
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
	w.Header().Add("Link", fmt.Sprintf(`<https://%s%s?max_id=%d>; rel="next", <https://%s%s?min_id=%d>; rel="prev"`, r.Host, r.URL.Path, oldest, r.Host, r.URL.Path, newest))
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
