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

func FavouritesCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var status models.Status
	query := env.DB.Joins("Actor").Scopes(models.PreloadStatus, models.PreloadReaction(user.Actor))
	if err := query.Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	reaction, err := models.NewReactions(env.DB).Favourite(&status, user.Actor)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(reaction.Status))
}

func FavouritesDestroy(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var status models.Status
	query := env.DB.Joins("Actor").Scopes(models.PreloadStatus, models.PreloadReaction(user.Actor))
	if err := query.Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	reaction, err := models.NewReactions(env.DB).Unfavourite(&status, user.Actor)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(reaction.Status))
}

func FavouritesIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var favourited []*models.Status
	query := env.DB.Joins("JOIN reactions ON reactions.status_id = statuses.id and reactions.actor_id = ? and reactions.favourited = ?", user.Actor.ID, true)
	query = query.Preload("Actor")
	query = query.Scopes(models.PaginateStatuses(r), models.PreloadStatus, models.PreloadReaction(user.Actor))
	if err := query.Find(&favourited).Error; err != nil {
		return err
	}

	if len(favourited) > 0 {
		linkHeader(w, r, favourited[0].ID, favourited[len(favourited)-1].ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(favourited, serialise.Status))
}
