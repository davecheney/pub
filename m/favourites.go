package m

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
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
	var status Status
	if err := f.service.db.Joins("Account").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := f.service.db.Model(user).Association("Favourites").Append(&status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.FavouritesCount++
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, status.serialize())
}

func (f *Favourites) Destroy(w http.ResponseWriter, req *http.Request) {
	user, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status Status
	if err := f.service.db.Joins("Account").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := f.service.db.Model(user).Association("Favourites").Delete(&status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.FavouritesCount--
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, status.serialize())
}

func (f *Favourites) Show(w http.ResponseWriter, req *http.Request) {
	_, err := f.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status Status
	if err := f.service.db.Joins("Account").First(&status, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var favs []Account
	if err := f.service.db.Model(&status).Association("Favourites").Find(&favs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []interface{}
	for _, fav := range favs {
		resp = append(resp, fav.serialize())
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, resp)
}
