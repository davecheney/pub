package mastodon

import (
	"fmt"
	"net/http"
	"net/url"
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

func (svc *Service) WellknownWebfinger(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	u, err := url.Parse(resource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if u.Scheme != "acct" {
		http.Error(w, "invalid scheme", http.StatusBadRequest)
		return
	}
	parts := strings.Split(u.Opaque, "@")
	if len(parts) != 2 {
		http.Error(w, "invalid resource", http.StatusBadRequest)
		return
	}
	username, domain := parts[0], parts[1]
	var account Account
	if err := svc.db.Where("username = ? AND domain = ?", username, domain).First(&account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	webfinger := fmt.Sprintf("https://%s/@%s", account.Domain, account.Username)
	self := fmt.Sprintf("https://%s/users/%s", account.Domain, account.Username)
	w.Header().Set("Content-Type", "application/jrd+json")
	json.MarshalFull(w, map[string]any{
		"subject": "acct:" + account.Acct(),
		"aliases": []string{webfinger, self},
		"links": []map[string]any{
			{
				"rel":  "http://webfinger.net/rel/profile-page",
				"type": "text/html",
				"href": webfinger,
			},
			{
				"rel":  "self",
				"type": "application/activity+json",
				"href": self,
			},
			{
				"rel":      "http://ostatus.org/schema/1.0/subscribe",
				"template": fmt.Sprintf("https://%s/authorize_interaction?uri={uri}", account.Domain),
			},
		},
	})
}
