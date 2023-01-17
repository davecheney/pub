package mastodon

import (
	"net/http"
	"strconv"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func RelationshipsShow(env *Env, w http.ResponseWriter, req *http.Request) error {
	user, err := env.authenticate(req)
	if err != nil {
		return err
	}
	targets := req.URL.Query()["id"]
	targets = append(targets, req.URL.Query()["id[]"]...)
	serialise := Serialiser{req: req}
	var resp []any
	for _, target := range targets {
		id, err := strconv.ParseUint(target, 10, 64)
		if err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
		tid := snowflake.ID(id)
		var rel models.Relationship
		if err := env.DB.Preload("Target").FirstOrCreate(&rel, models.Relationship{ActorID: user.Actor.ID, TargetID: tid}).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.Error(http.StatusNotFound, err)
			}
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
