package mastodon

import (
	"net/http"

	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

type Directory struct {
	service *Service
}

func (d *Directory) Index(w http.ResponseWriter, r *http.Request) {
	var actors []models.Actor
	query := d.service.DB().Scopes(models.PaginateActors(r), isLocal(r))
	if err := query.Find(&actors).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, a := range actors {
		resp = append(resp, serialiseAccount(&a))
	}
	toJSON(w, resp)
}

func isLocal(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.URL.Query().Get("local") != "" {
			return db.Where("domain = ?", r.Host)
		}
		return db
	}
}
