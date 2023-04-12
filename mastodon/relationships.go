package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func RelationshipsShow(env *Env, w http.ResponseWriter, req *http.Request) error {
	user, err := env.authenticate(req)
	if err != nil {
		return err
	}
	var params struct {
		ID  snowflake.ID   `schema:"id"`
		IDs []snowflake.ID `schema:"id[]"`
	}
	if err := httpx.Params(req, &params); err != nil {
		return err
	}
	serialise := Serialiser{req: req}
	var resp []any
	for _, tid := range append([]snowflake.ID{params.ID}, params.IDs...) {
		var rel models.Relationship
		if err := env.DB.Preload("Target").FirstOrInit(&rel, models.Relationship{ActorID: user.Actor.ID, TargetID: tid}).Error; err != nil {
			return err
		}
		resp = append(resp, serialise.Relationship(&rel))
	}
	return to.JSON(w, resp)
}

func RelationshipsCreate(env *Env, w http.ResponseWriter, req *http.Request) error {
	user, err := env.authenticate(req)
	if err != nil {
		return err
	}
	var target models.Actor
	if err := env.DB.First(&target, chi.URLParam(req, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	rel, err := models.NewRelationships(env.DB).Follow(user.Actor, &target)
	if err != nil {
		return err

	}
	serialise := Serialiser{req: req}
	return to.JSON(w, serialise.Relationship(rel))
}

func RelationshipsDestroy(env *Env, w http.ResponseWriter, req *http.Request) error {
	user, err := env.authenticate(req)
	if err != nil {
		return err
	}
	var target models.Actor
	if err := env.DB.First(&target, chi.URLParam(req, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	rel, err := models.NewRelationships(env.DB).Unfollow(user.Actor, &target)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: req}
	return to.JSON(w, serialise.Relationship(rel))
}
