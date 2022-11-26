package mastodon

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Status struct {
	gorm.Model
	AccountID          uint
	Account            *Account
	InReplyToID        *uint
	InReplyToAccountID *uint
	Sensitive          bool
	SpoilerText        string
	Visibility         string
	Language           string
	URI                string `gorm:"uniqueIndex"`
	RepliesCount       int
	ReblogsCount       int
	FavouritesCount    int
	Content            string
}

func (s *Status) serialize() map[string]any {
	return map[string]any{
		"id":                     strconv.Itoa(int(s.ID)),
		"created_at":             s.CreatedAt.UTC().Format("2006-01-02T15:04:05.006Z"),
		"in_reply_to_id":         s.InReplyToID,
		"in_reply_to_account_id": s.InReplyToAccountID,
		"sensitive":              s.Sensitive,
		"spoiler_text":           s.SpoilerText,
		"visibility":             s.Visibility,
		"language":               s.Language,
		"uri":                    fmt.Sprintf("https://cheney.net/users/%s/statuses/%d", s.Account.Username, s.ID),
		"url":                    fmt.Sprintf("https://cheney.net/@%s/%d", s.Account.Username, s.ID),
		"replies_count":          s.RepliesCount,
		"reblogs_count":          s.ReblogsCount,
		"favourites_count":       s.FavouritesCount,
		"favourited":             false,
		"reblogged":              false,
		"muted":                  false,
		"bookmarked":             false,
		"content":                s.Content,
		"reblog":                 nil,
		"application":            nil,
		"account":                s.Account.serialize(),
		"media_attachments":      []map[string]any{},
		"mentions":               []map[string]any{},
		"tags":                   []map[string]any{},
		"emojis":                 []map[string]any{},
		"card":                   nil,
		"poll":                   nil,
	}
}

type Statuses struct {
	db *gorm.DB
}

func NewStatuses(db *gorm.DB) *Statuses {
	return &Statuses{db: db}
}

func (s *Statuses) Create(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := s.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var toot struct {
		Status      string     `json:"status"`
		InReplyToID *uint      `json:"in_reply_to_id,string"`
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

	status := &Status{
		Account:     token.Account,
		AccountID:   token.AccountID,
		Sensitive:   toot.Sensitive,
		SpoilerText: toot.SpoilerText,
		Visibility:  toot.Visibility,
		Language:    toot.Language,
		Content:     toot.Status,
	}
	if err := s.db.Create(status).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	token.Account.LastStatusAt = time.Now()
	token.Account.StatusesCount++
	if err := s.db.Save(&token.Account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, status.serialize())
}
