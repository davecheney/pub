package wellknown

import (
	"fmt"
	"net/http"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/internal/webfinger"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

func WebfingerShow(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	acct, err := webfinger.Parse(r.URL.Query().Get("resource"))
	if err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}
	var actor models.Actor
	if err := env.DB.First(&actor, "name = ? AND domain = ?", acct.User, r.Host).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	self := acct.ID()
	return to.JSON(w, map[string]any{
		"subject": acct.String(),
		"aliases": []string{
			self,
		},
		"links": []map[string]any{
			{
				"rel":  "http://webfinger.net/rel/profile-page",
				"type": "text/html",
				"href": acct.Webfinger(),
			},
			{
				"rel":  "self",
				"type": "application/activity+json",
				"href": self,
			},
			{
				"rel":      "http://ostatus.org/schema/1.0/subscribe",
				"template": fmt.Sprintf("https://%s/authorize_interaction?uri={uri}", actor.Domain),
			},
		},
	})
}
