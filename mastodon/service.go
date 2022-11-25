package mastodon

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
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

func (svc *Service) AppsCreate(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientName   string  `json:"client_name"`
		Website      *string `json:"website"`
		RedirectURIs string  `json:"redirect_uris"`
		Scopes       string  `json:"scopes"`
	}
	if err := json.UnmarshalFull(r.Body, &params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println("/api/v1/apps: params:", params)

	app := &Application{
		Name:         params.ClientName,
		Website:      params.Website,
		ClientID:     uuid.New().String(),
		ClientSecret: uuid.New().String(),
		RedirectURI:  params.RedirectURIs,
		VapidKey:     "BCk-QqERU0q-CfYZjcuB6lnyyOYfJ2AifKqfeGIm7Z-HiTU5T9eTG5GxVA0_OH5mMlI4UkkDTpaZwozy0TzdZ2M=",
	}
	if err := svc.db.Create(app).Error; err != nil {
		http.Error(w, "failed to create application", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, map[string]any{
		"id":            strconv.Itoa(int(app.ID)),
		"name":          app.Name,
		"website":       app.Website,
		"redirect_uri":  app.RedirectURI,
		"client_id":     app.ClientID,
		"client_secret": app.ClientSecret,
		"vapid_key":     app.VapidKey,
	})
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
	account, err := svc.accounts().findByAcct(r.URL.Query().Get("resource"))
	if err != nil {
		log.Println("findAccountByAcct:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	webfinger := fmt.Sprintf("https://%s/@%s", account.Domain, account.Username)
	self := fmt.Sprintf("https://%s/users/%s", account.Domain, account.Username)
	w.Header().Set("Content-Type", "application/jrd+json")
	json.MarshalFull(w, map[string]any{
		"subject": account.Acct,
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

func (svc *Service) TimelinesHome(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := svc.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
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
