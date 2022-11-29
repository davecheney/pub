package m

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Timelines struct {
	db       *gorm.DB
	instance *Instance
}

func (t *Timelines) Index(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := t.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var statuses []Status
	scope := t.db.Scopes(t.paginate(r)).Joins("Account").Where("visibility = ?", "public")
	if err := scope.Order("statuses.id desc").Find(&statuses).Error; err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, status := range statuses {
		resp = append(resp, status.serialize())
	}

	w.Header().Set("Content-Type", "application/json")
	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/home?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/home?min_id=%d>; rel=\"prev\"", t.instance.Domain, statuses[len(statuses)-1].ID, t.instance.Domain, statuses[0].ID))
	}
	json.MarshalFull(w, resp)
}

func (t *Timelines) Public(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := t.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var statuses []Status
	scope := t.db.Scopes(t.paginate(r)).Preload("Account").Where("visibility = ?", "public")
	switch r.URL.Query().Get("local") {
	case "":
		scope = scope.Joins("Account")
	default:
		scope = scope.Joins("Account").Where("Account.instance_id = ?", t.instance.ID)
	}

	if err := scope.Order("statuses.id desc").Find(&statuses).Error; err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, status := range statuses {
		resp = append(resp, status.serialize())
	}

	w.Header().Set("Content-Type", "application/json")
	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/public?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/public?min_id=%d>; rel=\"prev\"", t.instance.Domain, statuses[len(statuses)-1].ID, t.instance.Domain, statuses[0].ID))
	}
	json.MarshalFull(w, resp)
}

func (t *Timelines) paginate(r *http.Request) func(db *gorm.DB) *gorm.DB {
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
