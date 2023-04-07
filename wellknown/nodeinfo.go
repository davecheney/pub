package wellknown

import (
	"fmt"
	"net/http"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

func NodeInfoIndex(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	return to.JSON(w, map[string]any{
		"links": []map[string]any{
			{
				"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.0",
				"href": fmt.Sprintf("https://%s/nodeinfo/2.0", r.Host),
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
	return to.JSON(w, serializeNodeInfo(&instance))
}

func serializeNodeInfo(i *models.Instance) map[string]any {
	return map[string]any{
		"version": "2.0", // https://github.com/jhass/nodeinfo/blob/main/schemas/2.0/schema.json
		"software": map[string]any{
			"name":    "https://github.com/davecheney/pub",
			"version": "0.0.0-devel",
		},
		"protocols": []string{
			"activitypub",
		},
		"services": map[string]any{
			"outbound": []any{},
			"inbound":  []any{},
		},
		"usage": map[string]any{
			"users": map[string]any{
				"total":          i.AccountsCount,
				"activeMonth":    0,
				"activeHalfyear": 0,
			},
			"localPosts": i.StatusesCount,
		},
		"openRegistrations": false,
		"metadata":          map[string]any{},
	}
}
