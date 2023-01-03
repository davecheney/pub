package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func MutesIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var mutes []*models.Relationship
	if err := env.DB.Joins("Target").Find(&mutes, "actor_id = ? and muting = true", user.Actor.ID).Error; err != nil {
		return err
	}

	return to.JSON(w, algorithms.Map(algorithms.Map(mutes, relationshipTarget), serialiseAccount))
}

func MutesCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var target models.Actor
	if err := env.DB.Take(&target, chi.URLParam(r, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	rel, err := models.NewRelationships(env.DB).Mute(user.Actor, &target)
	if err != nil {
		return err
	}
	return to.JSON(w, serialiseRelationship(rel))
}

func MutesDestroy(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var target models.Actor
	if err := env.DB.Take(&target, chi.URLParam(r, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	rel, err := models.NewRelationships(env.DB).Unmute(user.Actor, &target)
	if err != nil {
		return err
	}
	return to.JSON(w, serialiseRelationship(rel))
}
