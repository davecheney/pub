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

func FavouritesCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var status models.Status
	if err := env.DB.Joins("Actor").Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	reaction, err := models.NewReactions(env.DB).Favourite(&status, user.Actor)
	if err != nil {
		return err
	}
	status.Reaction = reaction
	status.FavouritesCount++
	return to.JSON(w, serialiseStatus(&status))
}

func FavouritesDestroy(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var status models.Status
	if err := env.DB.Joins("Actor").Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	reaction, err := models.NewReactions(env.DB).Unfavourite(&status, user.Actor)
	if err != nil {
		return err
	}
	status.Reaction = reaction
	status.FavouritesCount--
	return to.JSON(w, serialiseStatus(&status))
}

func FavouritesShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var reactions []*models.Reaction
	if err := env.DB.Joins("Actor").Where("status_id = ?", chi.URLParam(r, "id")).Find(&reactions).Error; err != nil {
		return err
	}

	return to.JSON(w, algorithms.Map(algorithms.Map(reactions, reactionActor), serialiseAccount))
}

func reactionActor(r *models.Reaction) *models.Actor { return r.Actor }
