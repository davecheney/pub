package m

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/m/internal/snowflake"
	"github.com/go-json-experiment/json"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Status struct {
	ID                 uint64 `gorm:"primaryKey;autoIncrement:false"`
	UpdatedAt          time.Time
	AccountID          uint
	Account            *Account
	ConversationID     uint
	InReplyToID        *uint64
	InReplyToAccountID *uint
	Sensitive          bool
	SpoilerText        string
	Visibility         string `gorm:"type:enum('public', 'unlisted', 'private', 'direct')"`
	Language           string
	URI                string `gorm:"uniqueIndex;size:128"`
	RepliesCount       int    `gorm:"not null;default:0"`
	ReblogsCount       int    `gorm:"not null;default:0"`
	FavouritesCount    int    `gorm:"not null;default:0"`
	Content            string

	FavouritedBy []Account `gorm:"many2many:favourites;"`
}

func (s *Status) AfterCreate(tx *gorm.DB) error {
	// update count of statuses on account
	var account Account
	if err := tx.Preload("Instance").Where("id = ?", s.AccountID).First(&account).Error; err != nil {
		return err
	}
	if err := account.updateStatusesCount(tx); err != nil {
		return err
	}
	return account.Instance.updateStatusesCount(tx)
}

func (s *Status) url() string {
	u, err := url.Parse(s.URI)
	if err != nil {
		return ""
	}
	id := path.Base(u.Path)
	return fmt.Sprintf("%s://%s/@%s/%s", u.Scheme, s.Account.Domain, s.Account.Username, id)
}

func (s *Status) serialize() map[string]any {
	return map[string]any{
		"id":                     strconv.Itoa(int(s.ID)),
		"created_at":             snowflake.IDToTime(s.ID).UTC().Format("2006-01-02T15:04:05.006Z"),
		"in_reply_to_id":         stringOrNull(s.InReplyToID),
		"in_reply_to_account_id": stringOrNull(s.InReplyToAccountID),
		"sensitive":              s.Sensitive,
		"spoiler_text":           s.SpoilerText,
		"visibility":             s.Visibility,
		"language":               "en", // s.Language,
		"uri":                    s.URI,
		"url":                    s.url(),
		"replies_count":          s.RepliesCount,
		"reblogs_count":          s.ReblogsCount,
		"favourites_count":       s.FavouritesCount,
		// "favourited":             false,
		// "reblogged":              false,
		// "muted":                  false,
		// "bookmarked":             false,
		"content":           s.Content,
		"text":              nil,
		"reblog":            nil,
		"application":       nil,
		"account":           s.Account.serialize(),
		"media_attachments": []map[string]any{},
		"mentions":          []map[string]any{},
		"tags":              []map[string]any{},
		"emojis":            []map[string]any{},
		"card":              nil,
		"poll":              nil,
	}
}

type Statuses struct {
	db      *gorm.DB
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

	createdAt := time.Now()
	id := snowflake.TimeToID(createdAt)
	status := &Status{
		ID:          id,
		AccountID:   account.ID,
		URI:         fmt.Sprintf("https://%s/@%s/%d", account.Domain, account.Username, id),
		Sensitive:   toot.Sensitive,
		SpoilerText: toot.SpoilerText,
		Visibility:  toot.Visibility,
		Language:    toot.Language,
		Content:     toot.Status,
	}
	if err := s.db.Create(status).Error; err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	account.LastStatusAt = createdAt
	if err := s.db.Save(account).Error; err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, status.serialize())
}

func (s *Statuses) Show(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var status Status
	if err := s.db.Preload("Account").Where("id = ?", id).First(&status).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, status.serialize())
}

type statuses struct {
	db      *gorm.DB
	service *Service
}

func (s *statuses) FindByURI(uri string) (*Status, error) {
	var status Status
	if err := s.db.Preload("Account").Where("uri = ?", uri).First(&status).Error; err != nil {
		return nil, err
	}
	return &status, nil
}

func (s *statuses) FindOrCreateStatus(uri string) (*Status, error) {
	status, err := s.FindByURI(uri)
	if err == nil {
		return status, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var obj map[string]interface{}
	if err := requests.URL(uri).Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).ToJSON(&obj).Fetch(context.Background()); err != nil {
		return nil, err
	}
	// json.MarshalOptions{}.MarshalFull(json.EncodeOptions{Indent: "  "}, os.Stdout, obj)
	// fmt.Println()

	if obj["type"] != "Note" {
		return nil, fmt.Errorf("unsupported type %q", obj["type"])
	}

	var visibility string
	for _, recipient := range obj["to"].([]interface{}) {
		if recipient == "https://www.w3.org/ns/activitystreams#Public" {
			visibility = "public"
			break
		}
	}
	switch visibility {
	case "public":
		// cool
	default:
		return nil, fmt.Errorf("unsupported visibility %q", visibility)
	}

	var inReplyTo *Status
	if inReplyToAtomUri, ok := obj["inReplyToAtomUri"].(string); ok {
		inReplyTo, err = s.FindOrCreateStatus(inReplyToAtomUri)
		if err != nil {
			return nil, err
		}
	}

	conversationID := uint(0)
	if inReplyTo != nil {
		conversationID = inReplyTo.ConversationID
	} else {
		conv := Conversation{
			Visibility: visibility,
		}
		if err := s.db.Create(&conv).Error; err != nil {
			return nil, err
		}
		conversationID = conv.ID
	}

	account, err := s.service.Accounts().FindOrCreateAccount(stringFromAny(obj["attributedTo"]))
	if err != nil {
		return nil, err
	}
	createdAt := timeFromAny(obj["published"])

	status = &Status{
		ID:             snowflake.TimeToID(createdAt),
		Account:        account,
		AccountID:      account.ID,
		ConversationID: conversationID,
		InReplyToID: func() *uint64 {
			if inReplyTo != nil {
				return &inReplyTo.ID
			}
			return nil
		}(),
		InReplyToAccountID: func() *uint {
			if inReplyTo != nil {
				return &inReplyTo.AccountID
			}
			return nil
		}(),
		Sensitive:   boolFromAny(obj["sensitive"]),
		SpoilerText: stringFromAny(obj["summary"]),
		Visibility:  "public",
		Language:    stringFromAny(obj["language"]),
		URI:         stringFromAny(obj["atomUri"]),
		Content:     stringFromAny(obj["content"]),
	}
	if err := s.db.Create(status).Error; err != nil {
		return nil, err
	}
	return status, nil
}

func timeFromAny(v any) time.Time {
	switch v := v.(type) {
	case string:
		t, _ := time.Parse(time.RFC3339, v)
		return t
	case time.Time:
		return v
	default:
		return time.Time{}
	}
}

type number interface {
	uint | uint64
}

func stringOrNull[T number](v *T) any {
	if v == nil {
		return nil
	}
	return strconv.Itoa(int(*v))
}

func contains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
