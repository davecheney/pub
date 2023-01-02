package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/to"
	"gorm.io/gorm"
)

type Directory struct {
	service *Service
}

func (d *Directory) Index(w http.ResponseWriter, r *http.Request) {
	var actors []*models.Actor
	query := d.service.db.Scopes(models.PaginateActors(r), isLocal(r))
	if err := query.Find(&actors).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	to.JSON(w, algorithms.Map(actors, serialiseAccount))
}

func isLocal(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.URL.Query().Get("local") != "" {
			return db.Where("domain = ?", r.Host)
		}
		return db
	}
}
