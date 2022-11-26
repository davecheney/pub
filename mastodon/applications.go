package mastodon

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Application struct {
	gorm.Model
	InstanceID   uint
	Instance     *Instance
	Name         string
	Website      *string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	VapidKey     string
}

type Applications struct {
	db       *gorm.DB
	instance *Instance
}

func NewApplications(db *gorm.DB, instance *Instance) *Applications {
	return &Applications{
		db:       db,
		instance: instance,
	}
}

func (a *Applications) New(w http.ResponseWriter, r *http.Request) {
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
		Instance:     a.instance,
		Name:         params.ClientName,
		Website:      params.Website,
		ClientID:     uuid.New().String(),
		ClientSecret: uuid.New().String(),
		RedirectURI:  params.RedirectURIs,
		VapidKey:     "BCk-QqERU0q-CfYZjcuB6lnyyOYfJ2AifKqfeGIm7Z-HiTU5T9eTG5GxVA0_OH5mMlI4UkkDTpaZwozy0TzdZ2M=",
	}
	if err := a.db.Create(app).Error; err != nil {
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
