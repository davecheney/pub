package mastodon

import (
	"net/http"

	"github.com/davecheney/m/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Blocks struct {
	service *Service
}

func (b *Blocks) Index(w http.ResponseWriter, r *http.Request) {
	user, err := b.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var blocks []models.Relationship
	if err := b.service.db.Joins("Target").Find(&blocks, "actor_id = ? and blocking = true", user.Actor.ID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	resp := []any{} // ensure we return an empty array, not null
	for _, a := range blocks {
		resp = append(resp, serialiseAccount(a.Target))
	}
	toJSON(w, resp)
}

func (b *Blocks) Create(w http.ResponseWriter, r *http.Request) {
	user, err := b.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target models.Actor
	if err := b.service.db.First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	rel, err := models.NewRelationships(b.service.db).Block(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialiseRelationship(rel))
}

func (b *Blocks) Destroy(w http.ResponseWriter, r *http.Request) {
	user, err := b.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target models.Actor
	if err := b.service.db.First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	rel, err := models.NewRelationships(b.service.db).Unblock(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialiseRelationship(rel))
}
