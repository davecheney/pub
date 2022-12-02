package m

import (
	"net/http"

	"github.com/go-json-experiment/json"
)

type Relationships struct {
	service *Service
}

func (r *Relationships) Show(w http.ResponseWriter, req *http.Request) {
	_, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id := req.URL.Query().Get("id")
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, []map[string]interface{}{{
		"id":                   id,
		"following":            false,
		"showing_reblogs":      true,
		"notifying":            false,
		"followed_by":          true,
		"blocking":             false,
		"blocked_by":           false,
		"muting":               false,
		"muting_notifications": false,
		"requested":            false,
		"domain_blocking":      false,
		"endorsed":             false,
		"note":                 "",
	}})
}
