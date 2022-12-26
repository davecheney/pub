package mastodon

import (
	"fmt"
	"net/http"

	"github.com/davecheney/m/internal/activitypub"
	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Relationships struct {
	service *Service
}

func (r *Relationships) Show(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	targets := req.URL.Query()["id"]
	targets = append(targets, req.URL.Query()["id[]"]...)
	fmt.Println("relationships show: targets: ", targets)
	var rels []m.Relationship
	if err := r.service.DB().Preload("Target").Find(&rels, "actor_id = ? and target_id IN (?) ", user.Actor.ID, targets).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []any
	for _, rel := range rels {
		resp = append(resp, serializeRelationship(&rel))
	}
	toJSON(w, resp)
}

func (r *Relationships) Create(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target m.Actor
	if err := r.service.DB().First(&target, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	err = r.sendFollowRequest(user, &target)
	if err != nil {
		fmt.Println("sendFollowRequest failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := m.NewService(r.service.DB())
	rel, err := svc.Relationships().Follow(user.Actor, &target)
	toJSON(w, serializeRelationship(rel))
}

func (r *Relationships) sendFollowRequest(account *m.Account, target *m.Actor) error {
	client, err := activitypub.NewClient(account.Actor.PublicKeyID(), account.PrivateKey)
	if err != nil {
		return err
	}
	return client.Follow(account.Actor.URI, target.URI)
}

func (r *Relationships) Destroy(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target m.Actor
	if err := r.service.DB().First(&target, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	err = r.sendUnfollowRequest(user, &target)
	if err != nil {
		fmt.Println("sendUnfollowRequest failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := m.NewService(r.service.DB())
	rel, err := svc.Relationships().Unfollow(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeRelationship(rel))
}

func (r *Relationships) sendUnfollowRequest(account *m.Account, target *m.Actor) error {
	// todo
	return nil
}

func serializeRelationship(rel *m.Relationship) map[string]any {
	return map[string]any{
		"id":                   toString(rel.Target.ID),
		"following":            rel.Following,
		"showing_reblogs":      true,  // todo
		"notifying":            false, // todo
		"followed_by":          rel.FollowedBy,
		"blocking":             rel.Blocking,
		"blocked_by":           rel.BlockedBy,
		"muting":               rel.Muting,
		"muting_notifications": false,
		"requested":            false,
		"domain_blocking":      false,
		"endorsed":             false,
		"note":                 rel.Target.Note,
	}
}
