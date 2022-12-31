package mastodon

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/davecheney/m/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Accounts struct {
	service *Service
}

func (a *Accounts) Show(w http.ResponseWriter, r *http.Request) {
	_, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var actor models.Actor
	if err := a.service.DB().First(&actor, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	toJSON(w, serialiseAccount(&actor))
}

func (a *Accounts) VerifyCredentials(w http.ResponseWriter, r *http.Request) {
	user, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	toJSON(w, serialiseCredentialAccount(user))
}

func (a *Accounts) StatusesShow(w http.ResponseWriter, r *http.Request) {
	_, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	var statuses []models.Status
	tx := a.service.DB().Preload("Actor").Where("actor_id = ?", id)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 40 {
		limit = 20
	}
	tx = tx.Limit(limit)
	sinceID, _ := strconv.Atoi(r.URL.Query().Get("since_id"))
	if sinceID > 0 {
		tx = tx.Where("id > ?", sinceID)
	}
	if err := tx.Order("id desc").Find(&statuses).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []any
	for _, status := range statuses {
		resp = append(resp, serialiseStatus(&status))
	}
	toJSON(w, resp)
}

func (a *Accounts) FollowersShow(w http.ResponseWriter, r *http.Request) {
	_, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var followers []models.Relationship
	if err := a.service.DB().Scopes(paginateRelationship(r)).Preload("Target").Where("actor_id = ? and followed_by = true", chi.URLParam(r, "id")).Find(&followers).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := []any{} // make sure this is a slice not null
	for _, follower := range followers {
		resp = append(resp, serialiseAccount(follower.Target))
	}
	if len(followers) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/accounts/%s/followers?max_id=%d>; rel=\"next\", <https://%s/api/v1/accounts/%s/followers?min_id=%d>; rel=\"prev\"", r.Host, chi.URLParam(r, "id"), followers[len(followers)-1].TargetID, r.Host, chi.URLParam(r, "id"), followers[0].TargetID))
	}
	toJSON(w, resp)
}

func (a *Accounts) FollowingShow(w http.ResponseWriter, r *http.Request) {
	_, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var following []models.Relationship
	if err := a.service.DB().Scopes(paginateRelationship(r)).Preload("Target").Where("actor_id = ? and following = true", chi.URLParam(r, "id")).Find(&following).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := []any{} // make sure this is a slice not null
	for _, f := range following {
		resp = append(resp, serialiseAccount(f.Target))
	}
	if len(following) > 0 {
		// TODO don't send if we're at the end of the list
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/accounts/%s/following?max_id=%d>; rel=\"next\", <https://%s/api/v1/accounts/%s/following?min_id=%d>; rel=\"prev\"", r.Host, chi.URLParam(r, "id"), following[len(following)-1].TargetID, r.Host, chi.URLParam(r, "id"), following[0].TargetID))
	}
	toJSON(w, resp)
}

func paginateRelationship(r *http.Request) func(db *gorm.DB) *gorm.DB {
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
			db = db.Where("target_id > ?", sinceID)
		}
		minID, _ := strconv.Atoi(r.URL.Query().Get("min_id"))
		if minID > 0 {
			db = db.Where("target_id > ?", minID)
		}
		maxID, _ := strconv.Atoi(r.URL.Query().Get("max_id"))
		if maxID > 0 {
			db = db.Where("target_id < ?", maxID)
		}
		return db.Order("target_id desc")
	}
}

func (a *Accounts) Update(w http.ResponseWriter, r *http.Request) {
	account, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Form.Get("display_name") != "" {
		account.Actor.DisplayName = r.Form.Get("display_name")
	}
	if r.Form.Get("note") != "" {
		account.Actor.Note = r.Form.Get("note")
	}

	if err := a.service.DB().Save(account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialiseAccount(account.Actor))
}
