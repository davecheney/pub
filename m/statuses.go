package m

import (
	stdjson "encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/davecheney/m/internal/activitypub"
	"github.com/davecheney/m/internal/snowflake"

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
