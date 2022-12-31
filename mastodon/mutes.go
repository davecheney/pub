package mastodon

import (
	"net/http"

	"github.com/davecheney/m/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Mutes struct {
	service *Service
}

func (svc *Mutes) Index(w http.ResponseWriter, r *http.Request) {
	user, err := svc.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var mutes []models.Relationship
	if err := svc.service.db.Joins("Target").Find(&mutes, "actor_id = ? and muting = true", user.Actor.ID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	var resp []any
	for _, a := range mutes {
		resp = append(resp, serialiseAccount(a.Target))
	}
	toJSON(w, resp)
}

func (svc *Mutes) Create(w http.ResponseWriter, r *http.Request) {
	user, err := svc.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target models.Actor
	if err := svc.service.db.First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	rel, err := models.NewRelationships(svc.service.db).Mute(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialiseRelationship(rel))
}

func (svc *Mutes) Destroy(w http.ResponseWriter, r *http.Request) {
	user, err := svc.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target models.Actor
	if err := svc.service.db.First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	rel, err := models.NewRelationships(svc.service.db).Unmute(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialiseRelationship(rel))
}
