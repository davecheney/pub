package mastodon

import (
	"net/http"
	"strconv"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
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
	resp := []any{} // ensure we return an array
	for _, target := range targets {
		id, err := strconv.ParseUint(target, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tid := snowflake.ID(id)
		var rel models.Relationship
		if err := r.service.db.Preload("Target").FirstOrCreate(&rel, models.Relationship{ActorID: user.Actor.ID, TargetID: tid}).Error; err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		resp = append(resp, serialiseRelationship(&rel))
	}
	to.JSON(w, resp)
}

func (r *Relationships) Create(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target models.Actor
	if err := r.service.db.First(&target, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	rel, err := models.NewRelationships(r.service.db).Follow(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	to.JSON(w, serialiseRelationship(rel))
}

func (r *Relationships) Destroy(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target models.Actor
	if err := r.service.db.First(&target, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	rel, err := models.NewRelationships(r.service.db).Unfollow(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	to.JSON(w, serialiseRelationship(rel))
}
