package wellknown

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func NodeInfoIndex(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("cache-control", "max-age=259200, public")
	return to.JSON(w, map[string]any{
		"links": []any{
			map[string]any{
				"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.0",
				"href": fmt.Sprintf("https://%s/nodeinfo/2.0", r.Host),
			},
			map[string]any{
				"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.1",
				"href": fmt.Sprintf("https://%s/nodeinfo/2.1", r.Host),
			},
		},
	})
}

func NodeInfoShow(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	var instance models.Instance
	if err := env.DB.Where("domain = ?", r.Host).First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	switch chi.URLParam(r, "version") {
	case "2.0":
		// https://github.com/jhass/nodeinfo/blob/main/schemas/2.0/schema.json
		w.Header().Set("cache-control", "max-age=259200, public")
		return to.JSON(w, map[string]any{
			"version": "2.0",
			"software": map[string]any{
				"name":    "https://github.com/davecheney/pub",
				"version": "0.0.0-devel",
			},
			"protocols":         protocols(),
			"services":          services(),
			"usage":             usage(),
			"openRegistrations": false,
			"metadata":          metadata(),
		})
	case "2.1":
		w.Header().Set("cache-control", "max-age=259200, public")
		return to.JSON(w, map[string]any{
			"version": "2.1",
			"software": map[string]any{
				"name":       "pub",
				"version":    "0.0.0-devel",
				"repository": "https://github.com/davecheney/pub",
			},
			"protocols":         protocols(),
			"services":          services(),
			"usage":             usage(),
			"openRegistrations": false,
			"metadata":          metadata(),
		})
	default:
		return httpx.Error(http.StatusNotFound, errors.New("unsupported version: "+chi.URLParam(r, "version")))
	}
}

func metadata() map[string]any {
	return map[string]any{}
}

func protocols() []any {
	return []any{
		"activitypub",
	}
}

func services() map[string]any {
	return map[string]any{
		"inbound":  []any{},
		"outbound": []any{},
	}
}

func usage() map[string]any {
	return map[string]any{
		"users": map[string]any{},
	}
}
