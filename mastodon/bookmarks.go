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

func BookmarksIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var bookmarked []*models.Status
	query := env.DB.Joins("JOIN reactions ON reactions.status_id = statuses.id and reactions.actor_id = ? and reactions.bookmarked = ?", user.Actor.ID, true)
	query = query.Preload("Actor")
	query = query.Scopes(models.PreloadStatus, models.PreloadReaction(user.Actor), models.PaginateStatuses(r))
	if err := query.Find(&bookmarked).Error; err != nil {
		return err
	}

	if len(bookmarked) > 0 {
		linkHeader(w, r, bookmarked[0].ID, bookmarked[len(bookmarked)-1].ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(bookmarked, serialise.Status))
}

func BookmarksCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
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
	reaction, err := models.NewReactions(env.DB).Bookmark(&status, user.Actor)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(reaction.Status))
}

func BookmarksDestroy(env *Env, w http.ResponseWriter, r *http.Request) error {
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
	reaction, err := models.NewReactions(env.DB).Unbookmark(&status, user.Actor)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(reaction.Status))
}
