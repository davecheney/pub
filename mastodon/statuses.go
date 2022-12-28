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
	user, err := s.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	actor := user.Actor
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
		conv, err = s.service.Service.Conversations().New(toot.Visibility)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	createdAt := time.Now()
	id := snowflake.TimeToID(createdAt)
	status := m.Status{
		ID:             id,
		ActorID:        actor.ID,
		Actor:          actor,
		ConversationID: conv.ID,
		InReplyToID:    toot.InReplyToID,
		URI:            fmt.Sprintf("https://%s/users/%s/%d", actor.Domain, actor.Name, id),
		Sensitive:      toot.Sensitive,
		SpoilerText:    toot.SpoilerText,
		Visibility:     toot.Visibility,
		Language:       toot.Language,
		Note:           toot.Status,
	}
	if err := s.service.DB().Create(&status).Error; err != nil {
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
	actor := account.Actor
	var status m.Status
	if err := s.service.DB().Joins("Actor").First(&status, chi.URLParam(r, "id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if status.ActorID != actor.ID {
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
	if err := s.service.DB().Joins("Actor").Preload("Reblog").Preload("Reblog.Actor").Preload("Attachments").First(&status, chi.URLParam(r, "id")).Error; err != nil {
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
		"created_at":             snowflake.IDToTime(s.ID).Round(time.Second).Format("2006-01-02T15:04:05.000Z"),
		"edited_at":              nil,
		"in_reply_to_id":         stringOrNull(s.InReplyToID),
		"in_reply_to_account_id": stringOrNull(s.InReplyToActorID),
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
			return fmt.Sprintf("%s://%s/@%s/%s", u.Scheme, s.Actor.Domain, s.Actor.Name, id)
		}(s),
		"replies_count":    s.RepliesCount,
		"reblogs_count":    s.ReblogsCount,
		"favourites_count": s.FavouritesCount,
		"favourited":       false, // todo
		"reblogged":        false, // todo
		"muted":            false, // todo
		"bookmarked":       false, // todo
		"content":          s.Note,
		"reblog": func(s *m.Status) any {
			if s.Reblog == nil {
				return nil
			}
			return serializeStatus(s.Reblog)
		}(s),
		"filtered":          []map[string]any{},
		"account":           serializeAccount(s.Actor),
		"media_attachments": serializeAttachments(s.Attachments),
		"mentions":          []map[string]any{},
		"tags":              []map[string]any{},
		"emojis":            []map[string]any{},
		"card":              nil,
		"poll":              nil,
	}
}

func serializeAttachments(atts []m.StatusAttachment) []map[string]any {
	var res []map[string]any
	for _, att := range atts {
		res = append(res, map[string]any{
			"id":          toString(att.ID),
			"type":        attachmentType(&att.Attachment),
			"url":         att.Attachment.URL,
			"preview_url": att.Attachment.URL,
			"remote_url":  nil,
			"description": att.Attachment.Name,
			"blurhash":    att.Attachment.Blurhash,
		})
	}
	return res
}

func attachmentType(att *m.Attachment) string {
	switch att.MediaType {
	case "image/jpeg":
		return "image"
	case "image/png":
		return "image"
	case "image/gif":
		return "image"
	case "video/mp4":
		return "video"
	case "video/webm":
		return "video"
	case "audio/mpeg":
		return "audio"
	case "audio/ogg":
		return "audio"
	default:
		return "unknown"
	}
}
