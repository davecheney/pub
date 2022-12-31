package mastodon

import (
	"net/http"

	"github.com/davecheney/m/internal/mime"
	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/internal/snowflake"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
)

type Applications struct {
	service *Service
}

func (a *Applications) Create(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientName   string  `json:"client_name"`
		Website      *string `json:"website"`
		RedirectURIs string  `json:"redirect_uris"`
		Scopes       string  `json:"scopes"`
	}
	switch mt := mime.MediaType(r); mt {
	case "application/x-www-form-urlencoded", "multipart/form-data":
		params.ClientName = r.PostFormValue("client_name")
		params.Website = ptr(r.PostFormValue("website"))
		params.RedirectURIs = r.PostFormValue("redirect_uris")
		params.Scopes = r.PostFormValue("scopes")
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported media type: "+mt, http.StatusUnsupportedMediaType)
		return
	}

	var instance models.Instance
	if err := a.service.db.First(&instance, "domain = ?", r.Host).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	app := &models.Application{
		ID:           snowflake.Now(),
		InstanceID:   instance.ID,
		Name:         params.ClientName,
		Website:      params.Website,
		ClientID:     uuid.New().String(),
		ClientSecret: uuid.New().String(),
		RedirectURI:  params.RedirectURIs,
		VapidKey:     "BCk-QqERU0q-CfYZjcuB6lnyyOYfJ2AifKqfeGIm7Z-HiTU5T9eTG5GxVA0_OH5mMlI4UkkDTpaZwozy0TzdZ2M=",
	}
	if err := a.service.db.Create(app).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	toJSON(w, map[string]any{
		"id":            toString(app.ID),
		"name":          app.Name,
		"website":       app.Website,
		"redirect_uri":  app.RedirectURI,
		"client_id":     app.ClientID,
		"client_secret": app.ClientSecret,
		"vapid_key":     app.VapidKey,
	})
}

func ptr[T any](v T) *T {
	return &v
}
