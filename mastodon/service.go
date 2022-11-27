package mastodon

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"github.com/go-json-experiment/json"
)

// Service implements a Mastodon service.
type Service struct {
	db *gorm.DB
}

// NewService returns a new instance of Service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (svc *Service) accounts() *Accounts {
	return &Accounts{db: svc.db}
}

func (svc *Service) AccountsFetch(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := svc.db.Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var account Account
	if err := svc.db.Where("id = ?", id).First(&account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, account.serialize())
}

func (svc *Service) AccountsStatusesFetch(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := svc.db.Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var account Account
	if err := svc.db.Where("id = ?", id).First(&account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var statuses []Status
	if err := svc.db.Preload("Account").Order("id desc").Limit(20).Find(&statuses).Error; err != nil {
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
