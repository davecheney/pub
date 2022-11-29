package m

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

// <https://hachyderm.io/api/v1/timelines/public?max_id=109419674974114267>; rel="next", <https://hachyderm.io/api/v1/timelines/public?min_id=109419683346234802>; rel="prev"

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
	URL                string
	RepliesCount       int
	ReblogsCount       int
	FavouritesCount    int
	Content            string
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
	u, _ := url.Parse(s.URI)
	id := path.Base(u.Path)
	return fmt.Sprintf("%s://%s/@%s/%s", u.Scheme, s.Account.Domain, s.Account.Username, id)
}

func (s *Status) serialize() map[string]any {
	return map[string]any{
		"id":                     strconv.Itoa(int(s.ID)),
		"created_at":             s.CreatedAt.UTC().Format("2006-01-02T15:04:05.006Z"),
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
	db       *gorm.DB
	instance *Instance
}

func NewStatuses(db *gorm.DB, instance *Instance) *Statuses {
	return &Statuses{db: db, instance: instance}
}

func (s *Statuses) accounts() *Accounts {
	return NewAccounts(s.db, s.instance)
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

func (s *Statuses) FindOrCreateStatus(uri string) (*Status, error) {
	var status Status
	err := s.db.Preload("Account").Where("uri = ?", uri).First(&status).Error
	if err == nil {
		return &status, nil
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

	var inReplyTo *Status
	if inReplyToAtomUri, ok := obj["inReplyToAtomUri"].(string); ok {
		inReplyTo, err = s.FindOrCreateStatus(inReplyToAtomUri)
		if err != nil {
			return nil, err
		}
	}

	account, err := s.accounts().FindOrCreateAccount(stringFromAny(obj["attributedTo"]))
	if err != nil {
		return nil, err
	}
	status = Status{
		Model: gorm.Model{
			CreatedAt: timeFromAny(obj["published"]),
		},
		Account:   account,
		AccountID: account.ID,
		InReplyToID: func() *uint {
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
		URL:         stringFromAny(obj["url"]),

		Content: stringFromAny(obj["content"]),
	}
	if err := s.db.Create(&status).Error; err != nil {
		return nil, err
	}
	return &status, nil
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

func stringOrNull(v *uint) any {
	if v == nil {
		return nil
	}
	return strconv.Itoa(int(*v))
}
