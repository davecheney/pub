package wellknown

import (
	"fmt"
	"io"
	"net/http"

	"github.com/davecheney/m/internal/webfinger"
	"github.com/davecheney/m/m"
	"github.com/go-json-experiment/json"
)

type Service struct {
	*m.Service
}

func NewService(service *m.Service) *Service {
	return &Service{
		Service: service,
	}
}

// NodeInfo returns a NodeInfo REST resource.
func (s *Service) NodeInfo() *NodeInfo {
	return &NodeInfo{
		service: s,
	}
}

func (w *Service) Webfinger(rw http.ResponseWriter, r *http.Request) {
	acct, err := webfinger.Parse(r.URL.Query().Get("resource"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var account m.Account
	if err := w.DB().Where("username = ? AND domain = ?", acct.User, r.Host).First(&account).Error; err != nil { // note, use the host from the request, not the acct
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	self := acct.ID()
	rw.Header().Set("Content-Type", "application/jrd+json")
	json.MarshalFull(rw, map[string]any{
		"subject": acct.String(),
		"aliases": []string{
			fmt.Sprintf("https://%s/@%s", account.Domain, account.Username),
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
				"template": fmt.Sprintf("https://%s/authorize_interaction?uri={uri}", account.Domain),
			},
		},
	})
}

func (w *Service) HostMeta(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/xrd+xml")
	io.WriteString(rw, `<?xml version="1.0" encoding="UTF-8"?>
		<XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
		<Subject>`+r.Host+`</Subject>
		<Link rel="lrdd" template="https://`+r.Host+`/.well-known/webfinger?resource={uri}"/>
		</XRD>`)
}

// toJSON writes the given object to the response body as JSON.
func toJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return json.MarshalFull(w, obj)
}
