package mastodon

import (
	"net/http"
	"time"

	"github.com/davecheney/m/m"
	"gorm.io/gorm"
)

// https://docs.joinmastodon.org/entities/V1_Filter/
type ClientFilter struct {
	gorm.Model
	AccountID    uint
	Account      *m.Account
	Phrase       string
	WholeWord    bool
	Context      []string `gorm:"serializer:json"`
	ExpiresAt    time.Time
	Irreversible bool
}

type Filters struct {
	service *Service
}

func (f *Filters) Index(w http.ResponseWriter, r *http.Request) {
	user, err := f.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var filters []ClientFilter
	if err := f.service.DB().Model(user).Association("Filters").Find(&filters); err != nil {
		if err == gorm.ErrRecordNotFound {
			toJSON(w, []any{})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, f := range filters {
		resp = append(resp, map[string]any{
			"id":           toString(f.ID),
			"phrase":       f.Phrase,
			"context":      f.Context,
			"whole_word":   false,
			"expires_at":   f.ExpiresAt.UTC().Format("2006-01-02T15:04:05.006Z"),
			"irreversible": true,
		})
	}
	toJSON(w, resp)
}
