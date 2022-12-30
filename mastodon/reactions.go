package mastodon

import (
	"fmt"
	"net/http"

	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/m"
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
	if err := f.service.DB().Joins("Actor").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	svc := m.NewService(f.service.DB())
	reaction, err := svc.Reactions().Favourite(&status, user.Actor)
	if err != nil {
		fmt.Println("favourite failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.Reaction = reaction
	status.FavouritesCount++
	toJSON(w, serializeStatus(&status))
}

func (f *Favourites) Destroy(w http.ResponseWriter, req *http.Request) {
	user, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status models.Status
	if err := f.service.DB().Joins("Actor").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	svc := m.NewService(f.service.DB())
	reaction, err := svc.Reactions().Unfavourite(&status, user.Actor)
	if err != nil {
		fmt.Println("unfavourite failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.Reaction = reaction
	status.FavouritesCount--
	toJSON(w, serializeStatus(&status))
}

func (f *Favourites) Show(w http.ResponseWriter, req *http.Request) {
	_, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var reactions []models.Reaction
	if err := f.service.DB().Preload("Actor").Where("status_id = ?", chi.URLParam(req, "id")).Find(&reactions).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var resp []interface{}
	for _, fav := range reactions {
		resp = append(resp, serializeAccount(fav.Actor))
	}
	toJSON(w, resp)
}
