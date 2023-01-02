package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Favourites struct {
	service *Service
}

func (f *Favourites) Create(w http.ResponseWriter, req *http.Request) {
	user, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status models.Status
	if err := f.service.db.Joins("Actor").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	reaction, err := models.NewReactions(f.service.db).Favourite(&status, user.Actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.Reaction = reaction
	status.FavouritesCount++
	toJSON(w, serialiseStatus(&status))
}

func (f *Favourites) Destroy(w http.ResponseWriter, req *http.Request) {
	user, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status models.Status
	if err := f.service.db.Joins("Actor").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	reaction, err := models.NewReactions(f.service.db).Unfavourite(&status, user.Actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.Reaction = reaction
	status.FavouritesCount--
	toJSON(w, serialiseStatus(&status))
}

func (f *Favourites) Show(w http.ResponseWriter, req *http.Request) {
	_, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var reactions []models.Reaction
	if err := f.service.db.Preload("Actor").Where("status_id = ?", chi.URLParam(req, "id")).Find(&reactions).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	resp := []any{} // ensure we return an empty array, not null
	for _, fav := range reactions {
		resp = append(resp, serialiseAccount(fav.Actor))
	}
	toJSON(w, resp)
}
