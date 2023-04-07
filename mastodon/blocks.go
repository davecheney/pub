package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func BlocksIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var blocks []*models.Relationship
	query := env.DB.Joins("Target").Scopes(models.PaginateRelationship(r))
	if err := query.Find(&blocks, "actor_id = ? and blocking = true", user.Actor.ID).Error; err != nil {
		return err
	}

	if len(blocks) > 0 {
		linkHeader(w, r, blocks[0].Target.ID, blocks[len(blocks)-1].Target.ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(algorithms.Map(blocks, relationshipTarget), serialise.Account))
}

func BlocksCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
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
	rel, err := models.NewRelationships(env.DB).Block(user.Actor, &target)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Relationship(rel))
}

func BlocksDestroy(env *Env, w http.ResponseWriter, r *http.Request) error {
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
	rel, err := models.NewRelationships(env.DB).Unblock(user.Actor, &target)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Relationship(rel))
}
