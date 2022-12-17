package wellknown

import (
	"fmt"
	"net/http"

	"github.com/davecheney/m/internal/webfinger"
	"github.com/go-json-experiment/json"
)

type Webfinger struct {
	service *Service
}

func (w *Webfinger) Show(rw http.ResponseWriter, r *http.Request) {
	acct, err := webfinger.Parse(r.URL.Query().Get("resource"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	actor, err := w.service.Actors().Find(acct.User, r.Host) // note, use the host from the request, not the acct
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}
	self := acct.ID()
	rw.Header().Set("Content-Type", "application/jrd+json")
	json.MarshalFull(rw, map[string]any{
		"subject": acct.String(),
		"aliases": []string{
			fmt.Sprintf("https://%s/@%s", actor.Domain, actor.Name),
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
