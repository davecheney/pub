package mastodon

import (
	"fmt"
	"net/http"

	"github.com/davecheney/m/internal/activitypub"
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
	account, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var actor m.Actor
	if err := r.service.DB().First(&actor, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	err = r.sendFollowRequest(account, &actor)
	if err != nil {
		fmt.Println("sendFollowRequest failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := r.service.DB().Model(account.Actor).Association("Following").Append(&actor); err != nil {
		fmt.Println("append failed", err)
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

func (r *Relationships) sendFollowRequest(account *m.Account, target *m.Actor) error {
	client, err := activitypub.NewClient(account.Actor.PublicKeyID(), account.PrivateKey)
	if err != nil {
		return err
	}
	return client.Follow(account.Actor.URI, target.URI)
}

func (r *Relationships) Destroy(w http.ResponseWriter, req *http.Request) {
	account, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var actor m.Actor
	if err := r.service.DB().First(&actor, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := r.service.DB().Model(account.Actor).Association("Following").Delete(&actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeRelationship(&m.Relationship{
		Target: &actor,
	}))
}

func serializeRelationship(rel *m.Relationship) map[string]any {
	return map[string]any{
		"id":                   toString(rel.Target.ID),
		"following":            false,
		"showing_reblogs":      false, // todo
		"notifying":            false, // todo
		"followed_by":          false, // todo
		"blocking":             false,
		"blocked_by":           false,
		"muting":               rel.Muting,
		"muting_notifications": false,
		"requested":            false,
		"domain_blocking":      false,
		"endorsed":             false,
		"note":                 rel.Target.Note,
	}
}
