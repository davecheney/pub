package mastodon

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type WellKnown struct {
	db       *gorm.DB
	instance *Instance
}

func NewWellKnown(db *gorm.DB, instance *Instance) *WellKnown {
	return &WellKnown{
		db:       db,
		instance: instance,
	}
}

func (w *WellKnown) Webfinger(rw http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	u, err := url.Parse(resource)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if u.Scheme != "acct" {
		http.Error(rw, "invalid scheme", http.StatusBadRequest)
		return
	}
	parts := strings.Split(u.Opaque, "@")
	if len(parts) != 2 {
		http.Error(rw, "invalid resource", http.StatusBadRequest)
		return
	}
	username, domain := parts[0], parts[1]
	var account Account
	if err := w.db.Where("username = ? AND domain = ?", username, domain).First(&account).Error; err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	webfinger := fmt.Sprintf("https://%s/@%s", account.Domain, account.Username)
	self := fmt.Sprintf("https://%s/users/%s", account.Domain, account.Username)
	rw.Header().Set("Content-Type", "application/jrd+json")
	json.MarshalFull(rw, map[string]any{
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

func (w *WellKnown) HostMeta(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/xrd+xml")
	io.WriteString(rw, `<?xml version="1.0" encoding="UTF-8"?>
		<XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
		<Subject>`+w.instance.Domain+`</Subject>
		<Link rel="lrdd" template="https://`+w.instance.Domain+`/.well-known/webfinger?resource={uri}"/>
		</XRD>`)
}
