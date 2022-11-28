package mastodon

import (
	"net/http"
	"strings"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Lists struct {
	db       *gorm.DB
	instance *Instance
}

func NewLists(db *gorm.DB, instance *Instance) *Lists {
	return &Lists{
		db:       db,
		instance: instance,
	}
}

func (l *Lists) Index(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := l.db.Preload("Account").Preload("Account.Lists").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var resp []any
	for _, list := range token.Account.Lists {
		resp = append(resp, list.serialize())
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, resp)
}
