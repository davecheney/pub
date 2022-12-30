package mastodon

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/internal/snowflake"
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

	var followers []models.Relationship
	if err := a.service.DB().Scopes(paginateRelationship(r)).Preload("Target").Where("actor_id = ? and followed_by = true", chi.URLParam(r, "id")).Find(&followers).Error; err != nil {
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
	var following []models.Relationship
	if err := a.service.DB().Scopes(paginateRelationship(r)).Preload("Target").Where("actor_id = ? and following = true", chi.URLParam(r, "id")).Find(&following).Error; err != nil {
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
	toJSON(w, serializeAccount(account.Actor))
}

type Account struct {
	ID             string           `json:"id"` // snowflake.ID `json:"id"`
	Username       string           `json:"username"`
	Acct           string           `json:"acct"`
	DisplayName    string           `json:"display_name"`
	Locked         bool             `json:"locked"`
	Bot            bool             `json:"bot"`
	Discoverable   *bool            `json:"discoverable"`
	Group          bool             `json:"group"`
	CreatedAt      string           `json:"created_at"`
	Note           string           `json:"note"`
	URL            string           `json:"url"`
	Avatar         string           `json:"avatar"`        // these four fields _cannot_ be blank
	AvatarStatic   string           `json:"avatar_static"` // if they are, various clients will consider the
	Header         string           `json:"header"`        // account to be invalid and ignore it or just go weird :grr:
	HeaderStatic   string           `json:"header_static"` // so they must be set to a default image.
	FollowersCount int32            `json:"followers_count"`
	FollowingCount int32            `json:"following_count"`
	StatusesCount  int32            `json:"statuses_count"`
	LastStatusAt   *string          `json:"last_status_at"`
	NoIndex        bool             `json:"noindex"` // default false
	Emojis         []map[string]any `json:"emojis"`
	Fields         []map[string]any `json:"fields"`
}

type CredentialAccount struct {
	*Account
	Source Source `json:"source"`
	Role   *Role  `json:"role,omitempty"`
}

type Role struct {
	ID          uint32 `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Position    int32  `json:"position"`
	Permissions uint32 `json:"permissions"`
	Highlighted bool   `json:"highlighted"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Source struct {
	Privacy             string           `json:"privacy"`
	Sensitive           bool             `json:"sensitive"`
	Language            string           `json:"language"`
	Note                string           `json:"note"`
	FollowRequestsCount int32            `json:"follow_requests_count"`
	Fields              []map[string]any `json:"fields"`
}

func serializeAccount(a *models.Actor) *Account {
	return &Account{
		ID:             toString(a.ID),
		Username:       a.Name,
		Acct:           a.Acct(),
		DisplayName:    a.DisplayName,
		Locked:         a.Locked,
		Bot:            a.IsBot(),
		Group:          a.IsGroup(),
		CreatedAt:      snowflake.ID(a.ID).ToTime().Round(time.Hour).Format("2006-01-02T00:00:00.000Z"),
		Note:           a.Note,
		URL:            fmt.Sprintf("https://%s/@%s", a.Domain, a.Name),
		Avatar:         a.Avatar,
		AvatarStatic:   a.Avatar,
		Header:         stringOrDefault(a.Header, "https://static.ma-cdn.net/headers/original/missing.png"),
		HeaderStatic:   stringOrDefault(a.Header, "https://static.ma-cdn.net/headers/original/missing.png"),
		FollowersCount: a.FollowersCount,
		FollowingCount: a.FollowingCount,
		StatusesCount:  a.StatusesCount,
		LastStatusAt: func() *string {
			if a.LastStatusAt.IsZero() {
				return nil
			}
			st := a.LastStatusAt.Format("2006-01-02")
			return &st
		}(),
		Emojis: make([]map[string]any, 0), // must be an empty array -- not null
		Fields: make([]map[string]any, 0), // ditto
	}
}

func serialiseCredentialAccount(a *models.Account) *CredentialAccount {
	ca := CredentialAccount{
		Account: serializeAccount(a.Actor),
		Source: Source{
			Privacy:   "public",
			Sensitive: false,
			Language:  "en",
			Note:      a.Actor.Note,
		},
	}
	if a.Role != nil {
		ca.Role = &Role{
			ID:          a.Role.ID,
			Name:        a.Role.Name,
			Color:       a.Role.Color,
			Position:    a.Role.Position,
			Permissions: a.Role.Permissions,
			Highlighted: a.Role.Highlighted,
			CreatedAt:   a.Role.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
			UpdatedAt:   a.Role.UpdatedAt.Format("2006-01-02T15:04:05.006Z"),
		}
	}
	return &ca
}
