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
	if err := b.service.DB().Joins("Target").Find(&blocks, "actor_id = ? and blocking = true", user.Actor.ID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	var resp []any
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
	if err := b.service.DB().First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	relationships := models.NewRelationships(b.service.DB())
	rel, err := relationships.Block(user.Actor, &target)
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
	if err := b.service.DB().First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	relationships := models.NewRelationships(b.service.DB())
	rel, err := relationships.Unblock(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialiseRelationship(rel))
}
