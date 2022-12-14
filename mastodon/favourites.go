package mastodon

import (
	"net/http"

	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
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
	var status m.Status
	if err := f.service.DB().Joins("Account").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := f.service.DB().Model(user).Association("Favourites").Append(&status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.FavouritesCount++
	toJSON(w, serializeStatus(&status))
}

func (f *Favourites) Destroy(w http.ResponseWriter, req *http.Request) {
	user, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status m.Status
	if err := f.service.DB().Joins("Account").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := f.service.DB().Model(user).Association("Favourites").Delete(&status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.FavouritesCount--
	toJSON(w, serializeStatus(&status))
}

func (f *Favourites) Show(w http.ResponseWriter, req *http.Request) {
	_, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status m.Status
	if err := f.service.DB().Joins("Account").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var favs []m.Actor
	if err := f.service.DB().Model(&status).Association("Favourites").Find(&favs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []interface{}
	for _, fav := range favs {
		resp = append(resp, serialize(&fav))
	}
	toJSON(w, resp)
}
