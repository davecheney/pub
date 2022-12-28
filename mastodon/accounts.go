package mastodon

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"github.com/davecheney/m/m"
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
	var actor m.Actor
	if err := a.service.DB().First(&actor, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	toJSON(w, serializeAccount(&actor))
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
	var statuses []m.Status
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
		resp = append(resp, serializeStatus(&status))
	}
	toJSON(w, resp)
}

func (a *Accounts) FollowersShow(w http.ResponseWriter, r *http.Request) {
	_, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var followers []m.Relationship
	if err := a.service.DB().Scopes(a.paginateRelationship(r)).Preload("Target").Where("actor_id = ? and followed_by = true", chi.URLParam(r, "id")).Find(&followers).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []any
	for _, follower := range followers {
		resp = append(resp, serializeAccount(follower.Target))
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
	var following []m.Relationship
	if err := a.service.DB().Scopes(a.paginateRelationship(r)).Preload("Target").Where("actor_id = ? and following = true", chi.URLParam(r, "id")).Find(&following).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, f := range following {
		resp = append(resp, serializeAccount(f.Target))
	}
	if len(following) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/accounts/%s/following?max_id=%d>; rel=\"next\", <https://%s/api/v1/accounts/%s/following?min_id=%d>; rel=\"prev\"", r.Host, chi.URLParam(r, "id"), following[len(following)-1].TargetID, r.Host, chi.URLParam(r, "id"), following[0].TargetID))
	}
	toJSON(w, resp)
}

func (a *Accounts) paginateRelationship(r *http.Request) func(db *gorm.DB) *gorm.DB {
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
	toJSON(w, serializeAccount(account.Actor))
}

func serializeAccount(a *m.Actor) map[string]any {
	return map[string]any{
		"id":       toString(a.ID),
		"username": a.Name,
		"acct": func(a *m.Actor) string {
			if a.Type == "LocalPerson" {
				return a.Name
			}
			return fmt.Sprintf("%s@%s", a.Name, a.Domain)
		}(a),
		"display_name":    a.DisplayName,
		"locked":          a.Locked,
		"bot":             a.Type == "Person",
		"discoverable":    true,
		"group":           a.Type == "Group",
		"created_at":      snowflake.IDToTime(a.ID).Round(time.Hour).Format("2006-01-02T00:00:00.000Z"),
		"note":            a.Note,
		"url":             fmt.Sprintf("https://%s/@%s", a.Domain, a.Name),
		"avatar":          stringOrDefault(a.Avatar, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"avatar_static":   stringOrDefault(a.Avatar, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"header":          stringOrDefault(a.Header, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"header_static":   stringOrDefault(a.Header, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"followers_count": a.FollowersCount,
		"following_count": a.FollowingCount,
		"statuses_count":  a.StatusesCount,
		"last_status_at": func(a *m.Actor) any {
			if a.LastStatusAt.IsZero() {
				return nil
			}
			return a.LastStatusAt.Format("2006-01-02")
		}(a),
		"noindex": false, // todo
		"emojis":  []map[string]any{},
		"fields":  []map[string]any{},
	}
}

func serialiseCredentialAccount(a *m.Account) map[string]any {
	m := serializeAccount(a.Actor)
	m["source"] = map[string]any{
		"privacy":               "public",
		"sensitive":             false,
		"language":              "en",
		"note":                  a.Actor.Note,
		"fields":                []map[string]any{},
		"follow_requests_count": 0,
	}
	if a.Role != nil {
		m["role"] = serialiseRole(a.Role)
	}
	return m
}

func serialiseRole(role *m.AccountRole) map[string]any {
	return map[string]any{
		"id":          role.ID,
		"name":        role.Name,
		"color":       role.Color,
		"position":    role.Position,
		"permissions": role.Permissions,
		"highlighted": role.Highlighted,
		"created_at":  role.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
		"updated_at":  role.UpdatedAt.Format("2006-01-02T15:04:05.006Z"),
	}
}
