package mastodon

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

type Timelines struct {
	service *Service
}

func (t *Timelines) Home(w http.ResponseWriter, r *http.Request) {
	user, err := t.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var followingIDs []int64
	if err := t.service.DB().Model(&models.Relationship{ActorID: user.Actor.ID}).Where("following = true").Pluck("target_id", &followingIDs).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	followingIDs = append(followingIDs, int64(user.ID))

	var statuses []models.Status
	scope := t.service.DB().Scopes(paginateStatuses(r)).Where("(actor_id IN (?) AND in_reply_to_actor_id is null) or (actor_id in (?) and in_reply_to_actor_id IN (?))", followingIDs, followingIDs, followingIDs)
	scope = scope.Joins("Actor").Preload("Reblog").Preload("Reblog.Actor").Preload("Attachments").Preload("Reaction", "actor_id = ?", user.Actor.ID)
	if err := scope.Find(&statuses).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []any
	for _, status := range statuses {
		resp = append(resp, serialiseStatus(&status))
	}
	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/home?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/home?min_id=%d>; rel=\"prev\"", r.Host, statuses[len(statuses)-1].ID, r.Host, statuses[0].ID))
	}
	toJSON(w, resp)
}

func (t *Timelines) Public(w http.ResponseWriter, r *http.Request) {
	user, err := t.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var statuses []models.Status
	scope := t.service.DB().Scopes(paginateStatuses(r)).Where("visibility = ? and reblog_id is null and in_reply_to_id is null", "public")
	switch r.URL.Query().Get("local") {
	case "true":
		scope = scope.Joins("Actor").Where("Actor.domain = ?", r.Host)
	default:
		scope = scope.Joins("Actor")
	}
	scope = scope.Preload("Attachments").Preload("Reaction", "actor_id = ?", user.Actor.ID)
	if err := scope.Find(&statuses).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, status := range statuses {
		resp = append(resp, serialiseStatus(&status))
	}
	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/public?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/public?min_id=%d>; rel=\"prev\"", r.Host, statuses[len(statuses)-1].ID, r.Host, statuses[0].ID))
	}
	toJSON(w, resp)
}

func paginateStatuses(r *http.Request) func(db *gorm.DB) *gorm.DB {
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
		return db.Order("statuses.id desc")
	}
}
