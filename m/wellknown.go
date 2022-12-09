package m

import (
	"fmt"
	"io"
	"net/http"

	"github.com/davecheney/m/internal/webfinger"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type WellKnown struct {
	db *gorm.DB
}

func (w *WellKnown) Webfinger(rw http.ResponseWriter, r *http.Request) {
	acct, err := webfinger.Parse(r.URL.Query().Get("resource"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var account Account
	if err := w.db.Where("username = ? AND domain = ?", acct.User, r.Host).First(&account).Error; err != nil { // note, use the host from the request, not the acct
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	self := acct.ID()
	rw.Header().Set("Content-Type", "application/jrd+json")
	json.MarshalFull(rw, map[string]any{
		"subject": fmt.Sprintf("acct:%s@%s", account.Username, account.Domain),
		"aliases": []string{self},
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
				"template": fmt.Sprintf("https://%s/authorize_interaction?uri={uri}", account.Domain),
			},
		},
	})
}

func (w *WellKnown) HostMeta(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/xrd+xml")
	io.WriteString(rw, `<?xml version="1.0" encoding="UTF-8"?>
		<XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
		<Subject>`+r.Host+`</Subject>
		<Link rel="lrdd" template="https://`+r.Host+`/.well-known/webfinger?resource={uri}"/>
		</XRD>`)
}
