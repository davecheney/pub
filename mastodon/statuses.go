package mastodon

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
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
		Status      string        `json:"status"`
		InReplyToID *snowflake.ID `json:"in_reply_to_id,string"`
		Sensitive   bool          `json:"sensitive"`
		SpoilerText string        `json:"spoiler_text"`
		Visibility  string        `json:"visibility"`
		Language    string        `json:"language"`
		ScheduledAt *time.Time    `json:"scheduled_at,omitempty"`
	}
	if err := json.UnmarshalFull(r.Body, &toot); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var conv *models.Conversation
	if toot.InReplyToID != nil {
		var parent models.Status
		if err := s.service.db.First(&parent, *toot.InReplyToID).Error; err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		conv, err = models.NewConversations(s.service.db).FindOrCreate(parent.ConversationID, toot.Visibility)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		conv, err = models.NewConversations(s.service.db).New(toot.Visibility)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	createdAt := time.Now()
	id := snowflake.TimeToID(createdAt)
	status := models.Status{
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
	if err := s.service.db.Create(&status).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	to.JSON(w, serialiseStatus(&status))
}

func (s *Statuses) Destroy(w http.ResponseWriter, r *http.Request) {
	account, err := s.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	actor := account.Actor
	var status models.Status
	if err := s.service.db.Joins("Actor").First(&status, chi.URLParam(r, "id")).Error; err != nil {
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
	if err := s.service.db.Delete(&status).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	to.JSON(w, serialiseStatus(&status))
}

func (s *Statuses) Show(w http.ResponseWriter, r *http.Request) {
	user, err := s.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var status models.Status
	query := s.service.db.Joins("Actor").Preload("Reblog").Preload("Reblog.Actor").Preload("Attachments").Preload("Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.First(&status, chi.URLParam(r, "id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	to.JSON(w, serialiseStatus(&status))
}
