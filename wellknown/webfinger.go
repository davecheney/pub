package wellknown

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/internal/webfinger"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

func WebfingerShow(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	resource := r.URL.Query().Get("resource")
	parts := strings.Split(resource, ":")
	if len(parts) != 2 {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("invalid resource %q", resource))
	}
	if parts[0] != "acct" {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("invalid resource %q", resource))
	}

	acct, err := webfinger.Parse(parts[1])
	if err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}
	actor, err := models.NewActors(env.DB).Find(acct.User, r.Host)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	w.Header().Set("cache-control", "max-age=3600, public")
	return to.JSON(w, map[string]any{
		"subject": fmt.Sprintf("acct:%s@%s", actor.Name, actor.Domain),
		"aliases": []string{
			actor.URI(),
		},
		"links": []any{
			map[string]any{
				"rel":  "self",
				"type": "application/activity+json",
				"href": actor.URI(),
			},
			map[string]any{
				"rel":      "http://ostatus.org/schema/1.0/subscribe",
				"template": fmt.Sprintf("https://%s/authorize_interaction?uri={uri}", actor.Domain),
			},
		},
	})
}
