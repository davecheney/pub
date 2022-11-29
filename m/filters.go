package m

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

// https://docs.joinmastodon.org/entities/V1_Filter/
type ClientFilter struct {
	gorm.Model
	AccountID    uint
	Account      *Account
	Phrase       string
	WholeWord    bool
	Context      []string `gorm:"serializer:json"`
	ExpiresAt    time.Time
	Irreversible bool
}

func (c *ClientFilter) serialize() map[string]any {
	return map[string]any{
		"id":           strconv.Itoa(int(c.ID)),
		"phrase":       c.Phrase,
		"context":      c.Context,
		"whole_word":   false,
		"expires_at":   c.ExpiresAt.UTC().Format("2006-01-02T15:04:05.006Z"),
		"irreversible": true,
	}
}

type Filters struct {
	db *gorm.DB
}

func NewFilters(db *gorm.DB) *Filters {
	return &Filters{
		db: db,
	}
}

func (f *Filters) Index(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	var token Token
	if err := f.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var filters []ClientFilter
	if err := f.db.Where("account_id = ?", token.AccountID).Find(&filters).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, filter := range filters {
		resp = append(resp, filter.serialize())
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, resp)
}
