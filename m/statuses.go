package m

import (
	stdjson "encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/internal/snowflake"
	"github.com/go-chi/chi/v5"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Status struct {
	ID                 uint64 `gorm:"primaryKey;autoIncrement:false"`
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
	AccountID          uint
	Account            *Account
	ConversationID     uint
	InReplyToID        *uint64
	InReplyToAccountID *uint
	Sensitive          bool
	SpoilerText        string
	Visibility         string `gorm:"type:enum('public', 'unlisted', 'private', 'direct', 'limited')"`
	Language           string
	URI                string `gorm:"uniqueIndex;size:128"`
	RepliesCount       int    `gorm:"not null;default:0"`
	ReblogsCount       int    `gorm:"not null;default:0"`
	FavouritesCount    int    `gorm:"not null;default:0"`
	Content            string

	FavouritedBy []Account `gorm:"many2many:account_favourites"`
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

	conv, err := s.service.conversations().FindConversationByStatusID(func(id *uint64) uint64 {
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
		conv = &Conversation{
			Visibility: toot.Visibility,
		}
		if err := s.db.Create(conv).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	createdAt := time.Now()
	id := snowflake.TimeToID(createdAt)
	status := Status{
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
	if err := s.db.Model(conv).Association("Statuses").Append(&status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, status.serialize())
}

func (s *Statuses) Destroy(w http.ResponseWriter, r *http.Request) {
	account, err := s.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var status Status
	if err := s.db.Where("statuses.id = ?", chi.URLParam(r, "id")).Joins("Account").First(&status).Error; err != nil {
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
	if err := s.db.Delete(&status).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, status.serialize())
}

func (s *Statuses) Show(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var status Status
	if err := s.db.Joins("Account").First(&status, id).Error; err != nil {
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

func (s *statuses) NewRemoteStatusFetcher() *RemoteStatusFetcher {
	return &RemoteStatusFetcher{
		service: s.service,
	}
}

type RemoteStatusFetcher struct {
	service *Service
}

func (f *RemoteStatusFetcher) Fetch(uri string) (*Status, error) {
	obj, err := f.fetch(uri)
	if err != nil {
		return nil, err
	}

	if obj["type"] != "Note" {
		return nil, fmt.Errorf("unsupported type %q", obj["type"])
	}

	var visibility string
	for _, recipient := range anyToSlice(obj["to"]) {
		switch recipient {
		case "https://www.w3.org/ns/activitystreams#Public":
			visibility = "public"
		case stringFromAny(obj["attributedTo"]) + "/followers":
			visibility = "limited"
		}
	}
	if visibility == "" {
		for _, recipient := range anyToSlice(obj["cc"]) {
			switch recipient {
			case "https://www.w3.org/ns/activitystreams#Public":
				visibility = "public"
			case stringFromAny(obj["attributedTo"]) + "/followers":
				visibility = "limited"
			}
		}
	}
	if visibility == "" {
		x, _ := stdjson.MarshalIndent(obj, "", "  ")
		return nil, fmt.Errorf("unsupported visibility %q: %s", visibility, x)
	}

	var inReplyTo *Status
	if inReplyToURI := stringFromAny(obj["inReplyTo"]); inReplyToURI != "" {
		inReplyTo, err = f.service.Statuses().FindOrCreate(inReplyToURI, f.Fetch)
		if err != nil {
			aerr := new(activitypub.Error)
			if errors.As(err, &aerr) && aerr.StatusCode != http.StatusNotFound {
				return nil, err
			}
			// 404 is fine, it just means the status is no longer available
		}
	}

	conversationID := uint(0)
	if inReplyTo != nil {
		conversationID = inReplyTo.ConversationID
	} else {
		conv := Conversation{
			Visibility: visibility,
		}
		if err := f.service.db.Create(&conv).Error; err != nil {
			return nil, err
		}
		conversationID = conv.ID
	}

	fetcher := f.service.Accounts().NewRemoteAccountFetcher()
	account, err := f.service.Accounts().FindOrCreate(stringFromAny(obj["attributedTo"]), fetcher.Fetch)
	if err != nil {
		return nil, err
	}
	createdAt := timeFromAny(obj["published"])

	return &Status{
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
		URI:         uri,
		Content:     stringFromAny(obj["content"]),
	}, nil
}

func (f *RemoteStatusFetcher) fetch(uri string) (map[string]interface{}, error) {
	// use admin account to sign the request
	signAs, err := f.service.Accounts().FindAdminAccount()
	if err != nil {
		return nil, err
	}
	c, err := activitypub.NewClient(signAs.PublicKeyID(), signAs.LocalAccount.PrivateKey)
	if err != nil {
		return nil, err
	}
	return c.Get(uri)
}

// FindOrCreate searches for a status by its URI. If the status is not found, it
// calls the given function to create a new status, stores that status in the
// database and returns it.
func (s *statuses) FindOrCreate(uri string, createFn func(string) (*Status, error)) (*Status, error) {
	status, err := s.FindByURI(uri)
	if err == nil {
		return status, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	status, err = createFn(uri)
	if err != nil {
		fmt.Println("findOrCreate: createFn:", err)
		return nil, err
	}
	if err := s.db.Create(&status).Error; err != nil {
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

func anyToSlice(v any) []any {
	switch v := v.(type) {
	case []any:
		return v
	default:
		return nil
	}
}
