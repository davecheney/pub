package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/google/uuid"
)

func AppsCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	var params struct {
		ClientName   string `json:"client_name" schema:"client_name,required"`
		Website      string `json:"website" schema:"website,required"`
		RedirectURIs string `json:"redirect_uris" schema:"redirect_uris,required"`
		Scopes       string `json:"scopes" schema:"scopes,required"`
	}
	if err := httpx.Params(r, &params); err != nil {
		return err
	}

	var instance models.Instance
	if err := env.DB.Take(&instance, "domain = ?", r.Host).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
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
		Scopes:       params.Scopes,
	}
	if err := env.DB.Create(app).Error; err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Application(app))
}
