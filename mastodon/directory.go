package mastodon

import (
	"net/http"
	"strconv"

	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

type Directory struct {
	service *Service
}

func (d *Directory) Index(w http.ResponseWriter, r *http.Request) {
	var actors []models.Actor
	query := d.service.DB().Scopes(paginateActors(r), isLocal(r))
	if err := query.Find(&actors).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, a := range actors {
		resp = append(resp, serializeAccount(&a))
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

func paginateActors(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()

		limit, _ := strconv.Atoi(q.Get("limit"))
		switch {
		case limit > 40:
			limit = 80
		case limit <= 0:
			limit = 20
		}
		db = db.Limit(limit)

		offset, _ := strconv.Atoi(q.Get("offset"))
		db = db.Offset(offset)

		switch q.Get("order") {
		case "new":
			db = db.Order("id desc")
		case "active":
			db = db.Order("last_status_at desc")
		}
		return db
	}
}
