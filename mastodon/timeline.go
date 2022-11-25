package mastodon

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Timeline struct {
	db *gorm.DB
}

func NewTimeline(db *gorm.DB) *Timeline {
	return &Timeline{
		db: db,
	}
}

func (t *Timeline) Index(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := t.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var statuses []Status
	if err := t.db.Scopes(t.paginate(r)).Preload("Account").Find(&statuses).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []any
	for _, status := range statuses {
		resp = append(resp, status.serialize())
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, resp)
}

func (t *Timeline) paginate(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		switch {
		case limit > 40:
			limit = 40
		case limit <= 0:
			limit = 20
		}
		return db.Limit(limit).Order("id desc")
	}
}
