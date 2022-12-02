package m

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

func (r *Relationships) Create(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(req, "id")
	var account Account
	if err := r.service.db.First(&account, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := r.service.db.Model(&user).Association("Following").Append(&account); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, map[string]interface{}{
		"id":                   strconv.Itoa(int(account.ID)),
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
		"note":                 account.Note,
	})
}

func (r *Relationships) Delete(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(req, "id")
	var account Account
	if err := r.service.db.First(&account, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := r.service.db.Model(&user).Association("Following").Delete(&account); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, map[string]interface{}{
		"id":                   strconv.Itoa(int(account.ID)),
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
		"note":                 account.Note,
	})
}
