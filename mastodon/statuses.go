package mastodon

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Statuses struct {
	service *Service
}

func (s *Statuses) Create(w http.ResponseWriter, r *http.Request) {
	account, err := s.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var toot struct {
		Status      string     `json:"status"`
		InReplyToID *uint64    `json:"in_reply_to_id,string"`
		Sensitive   bool       `json:"sensitive"`
		SpoilerText string     `json:"spoiler_text"`
		Visibility  string     `json:"visibility"`
		Language    string     `json:"language"`
		ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	}
	if err := json.UnmarshalFull(r.Body, &toot); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conv, err := s.service.Service.Conversations().FindConversationByStatusID(func(id *uint64) uint64 {
		if id == nil {
			return 0
		}
		return *id
	}(toot.InReplyToID))
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		conv = &m.Conversation{
			Visibility: toot.Visibility,
		}
		if err := s.service.DB().Create(conv).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	createdAt := time.Now()
	id := snowflake.TimeToID(createdAt)
	status := m.Status{
		ID:             id,
		AccountID:      account.ID,
		Account:        account,
		ConversationID: conv.ID,
		InReplyToID:    toot.InReplyToID,
		URI:            fmt.Sprintf("https://%s/users/%s/%d", account.Domain, account.Username, id),
		Sensitive:      toot.Sensitive,
		SpoilerText:    toot.SpoilerText,
		Visibility:     toot.Visibility,
		Language:       toot.Language,
		Content:        toot.Status,
	}
	if err := s.service.DB().Model(conv).Association("Statuses").Append(&status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.service.DB().Model(account).Association("Statuses").Append(&status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeStatus(&status))
}

func (s *Statuses) Destroy(w http.ResponseWriter, r *http.Request) {
	account, err := s.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var status m.Status
	if err := s.service.DB().Where("statuses.id = ?", chi.URLParam(r, "id")).Joins("Account").First(&status).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if status.AccountID != account.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := s.service.DB().Delete(&status).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeStatus(&status))
}

func (s *Statuses) Show(w http.ResponseWriter, r *http.Request) {
	_, err := s.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status m.Status
	if err := s.service.DB().Where("statuses.id = ?", chi.URLParam(r, "id")).Joins("Account").First(&status).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeStatus(&status))
}

func serializeStatus(s *m.Status) map[string]any {
	return map[string]any{
		"id":                     toString(s.ID),
		"created_at":             snowflake.IDToTime(s.ID).UTC().Format("2006-01-02T15:04:05.006Z"),
		"in_reply_to_id":         stringOrNull(s.InReplyToID),
		"in_reply_to_account_id": stringOrNull(s.InReplyToAccountID),
		"sensitive":              s.Sensitive,
		"spoiler_text":           s.SpoilerText,
		"visibility":             s.Visibility,
		"language":               "en", // s.Language,
		"uri":                    s.URI,
		"url": func(s *m.Status) string {
			u, err := url.Parse(s.URI)
			if err != nil {
				return ""
			}
			id := path.Base(u.Path)
			return fmt.Sprintf("%s://%s/@%s/%s", u.Scheme, s.Account.Domain, s.Account.Username, id)
		}(s),
		"replies_count":    s.RepliesCount,
		"reblogs_count":    s.ReblogsCount,
		"favourites_count": s.FavouritesCount,
		// "favourited":             false,
		// "reblogged":              false,
		// "muted":                  false,
		// "bookmarked":             false,
		"content":           s.Content,
		"text":              nil,
		"reblog":            nil,
		"application":       nil,
		"account":           serialize(s.Account),
		"media_attachments": []map[string]any{},
		"mentions":          []map[string]any{},
		"tags":              []map[string]any{},
		"emojis":            []map[string]any{},
		"card":              nil,
		"poll":              nil,
	}
}
