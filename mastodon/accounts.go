package mastodon

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
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
	var account m.Account
	if err := a.service.DB().First(&account, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	toJSON(w, serialize(&account))
}

func (a *Accounts) VerifyCredentials(w http.ResponseWriter, r *http.Request) {
	user, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	toJSON(w, serialize(user))
}

func (a *Accounts) StatusesShow(w http.ResponseWriter, r *http.Request) {
	_, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	var statuses []m.Status
	tx := a.service.DB().Preload("Account").Where("account_id = ?", id)
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
		account.DisplayName = r.Form.Get("display_name")
	}
	if r.Form.Get("note") != "" {
		account.Note = r.Form.Get("note")
	}

	if err := a.service.DB().Omit("Instance").Save(account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialize(account))
}

func serialize(a *m.Account) map[string]any {
	return map[string]any{
		"id":       strconv.Itoa(int(a.ID)),
		"username": a.Username,
		"acct": func(a *m.Account) string {
			if a.Local {
				return a.Username
			}
			return fmt.Sprintf("%s@%s", a.Username, a.Domain)
		}(a),
		"display_name":    a.DisplayName,
		"locked":          a.Locked,
		"bot":             a.Bot,
		"discoverable":    true,
		"group":           false, // todo
		"created_at":      a.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
		"note":            a.Note,
		"url":             a.URL(),
		"avatar":          stringOrDefault(a.Avatar, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"avatar_static":   stringOrDefault(a.Avatar, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"header":          stringOrDefault(a.Header, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"header_static":   stringOrDefault(a.Header, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"followers_count": a.FollowersCount,
		"following_count": a.FollowingCount,
		"statuses_count":  a.StatusesCount,
		"last_status_at":  a.LastStatusAt.Format("2006-01-02"),
		"noindex":         false, // todo
		"emojis":          []map[string]any{},
		"fields":          []map[string]any{},
	}
}
