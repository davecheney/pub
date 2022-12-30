package mastodon

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

type Conversations struct {
	service *Service
}

func (c *Conversations) Index(w http.ResponseWriter, r *http.Request) {
	_, err := c.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var statuses []models.Status
	scope := c.service.DB().Scopes(c.paginate(r)).Where("visibility = ?", "direct")
	switch r.URL.Query().Get("local") {
	case "":
		scope = scope.Joins("Actor")
	default:
		scope = scope.Joins("Actor").Where("Actor.domain = ?", r.Host)
	}

	if err := scope.Order("statuses.id desc").Find(&statuses).Error; err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, status := range statuses {
		resp = append(resp, serializeStatus(&status))
	}
	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/public?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/public?min_id=%d>; rel=\"prev\"", r.Host, statuses[len(statuses)-1].ID, r.Host, statuses[0].ID))
	}
	toJSON(w, resp)
}

func (c *Conversations) paginate(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()

		limit, _ := strconv.Atoi(q.Get("limit"))
		switch {
		case limit > 40:
			limit = 40
		case limit <= 0:
			limit = 20
		}
		db = db.Limit(limit)

		sinceID, _ := strconv.Atoi(r.URL.Query().Get("since_id"))
		if sinceID > 0 {
			db = db.Where("statuses.id > ?", sinceID)
		}
		minID, _ := strconv.Atoi(r.URL.Query().Get("min_id"))
		if minID > 0 {
			db = db.Where("statuses.id > ?", minID)
		}
		maxID, _ := strconv.Atoi(r.URL.Query().Get("max_id"))
		if maxID > 0 {
			db = db.Where("statuses.id < ?", maxID)
		}
		return db
	}
}
