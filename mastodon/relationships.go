package mastodon

import (
	"net/http"

	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
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
	toJSON(w, []map[string]interface{}{{
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

func (r *Relationships) Create(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var actor m.Actor
	if err := r.service.DB().First(&actor, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := r.service.DB().Model(&user).Association("Following").Append(&actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, map[string]interface{}{
		"id":                   toString(actor.ID),
		"following":            true,
		"showing_reblogs":      true,  // todo
		"notifying":            true,  // todo
		"followed_by":          false, // todo
		"blocking":             false,
		"blocked_by":           false,
		"muting":               false,
		"muting_notifications": false,
		"requested":            false,
		"domain_blocking":      false,
		"endorsed":             false,
		"note":                 actor.Note,
	})
}

func (r *Relationships) Destroy(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var actor m.Actor
	if err := r.service.DB().First(&actor, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := r.service.DB().Model(&user).Association("Following").Delete(&actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, map[string]interface{}{
		"id":                   toString(actor.ID),
		"following":            false,
		"showing_reblogs":      false, // todo
		"notifying":            false, // todo
		"followed_by":          false, // todo
		"blocking":             false,
		"blocked_by":           false,
		"muting":               false,
		"muting_notifications": false,
		"requested":            false,
		"domain_blocking":      false,
		"endorsed":             false,
		"note":                 actor.Note,
	})
}
