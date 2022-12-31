package m

import (
	stdjson "encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/davecheney/m/internal/activitypub"
	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/internal/snowflake"

	"gorm.io/gorm"
)

type statuses struct {
	db      *gorm.DB
	service *Service
}

func (s *statuses) FindByURI(uri string) (*models.Status, error) {
	var status models.Status
	if err := s.db.Preload("Actor").Where("uri = ?", uri).First(&status).Error; err != nil {
		return nil, err
	}
	return &status, nil
}

func (s *statuses) NewRemoteStatusFetcher(signAs *models.Account) *RemoteStatusFetcher {
	return &RemoteStatusFetcher{
		signAs:  signAs,
		service: s.service,
	}
}

type RemoteStatusFetcher struct {
	signAs  *models.Account
	service *Service
}

func (f *RemoteStatusFetcher) Fetch(uri string) (*models.Status, error) {
	obj, err := f.fetch(uri)
	if err != nil {
		return nil, err
	}

	typ := stringFromAny(obj["type"])
	switch typ {
	case "Note":
		// cool
	case "Question":
		// cool
	default:
		return nil, fmt.Errorf("unsupported type %q", typ)
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

	var inReplyTo *models.Status
	if inReplyToURI := stringFromAny(obj["inReplyTo"]); inReplyToURI != "" {
		inReplyTo, err = f.service.Statuses().FindOrCreate(inReplyToURI, f.Fetch)
		if err != nil {
			fmt.Println("inReplyToURI", inReplyToURI, err)
		}
	}

	f.service.Conversations()
	var conversationID uint32
	if inReplyTo != nil {
		conversationID = inReplyTo.ConversationID
	} else {
		conv := models.Conversation{
			Visibility: visibility,
		}
		if err := f.service.db.Create(&conv).Error; err != nil {
			return nil, err
		}
		conversationID = conv.ID
	}

	fetcher := f.service.Actors().NewRemoteActorFetcher(f.signAs)
	actor, err := f.service.Actors().FindOrCreate(stringFromAny(obj["attributedTo"]), fetcher.Fetch)
	if err != nil {
		return nil, err
	}
	createdAt := timeFromAny(obj["published"])

	st := &models.Status{
		ID:               snowflake.TimeToID(createdAt),
		ActorID:          actor.ID,
		ConversationID:   conversationID,
		InReplyToID:      inReplyToID(inReplyTo),
		InReplyToActorID: inReplyToActorID(inReplyTo),
		Sensitive:        boolFromAny(obj["sensitive"]),
		SpoilerText:      stringFromAny(obj["summary"]),
		Visibility:       "public",
		Language:         stringFromAny(obj["language"]),
		URI:              uri,
		Note:             stringFromAny(obj["content"]),
	}
	for _, att := range anyToSlice(obj["attachment"]) {
		at := mapFromAny(att)
		fmt.Println("attachment:", at)
		st.Attachments = append(st.Attachments, models.StatusAttachment{
			Attachment: models.Attachment{
				ID:        snowflake.Now(),
				MediaType: stringFromAny(at["mediaType"]),
				URL:       stringFromAny(at["url"]),
				Name:      stringFromAny(at["name"]),
				Width:     intFromAny(at["width"]),
				Height:    intFromAny(at["height"]),
				Blurhash:  stringFromAny(at["blurhash"]),
			},
		})
	}
	return st, nil
}

func inReplyToID(inReplyTo *models.Status) *snowflake.ID {
	if inReplyTo != nil {
		return &inReplyTo.ID
	}
	return nil
}

func inReplyToActorID(inReplyTo *models.Status) *snowflake.ID {
	if inReplyTo != nil {
		return &inReplyTo.ActorID
	}
	return nil
}

// // noteToStatus converts an ActivityPub note to a Status.
// func noteToStatus(note map[string]interface{}) (*Status, error) {
// 	createdAt := timeFromAny(note["published"])
// 	st := &Status{
// 		ID:             snowflake.TimeToID(createdAt),
// 		ActorID:        actor.ID,
// 		Actor:          actor,
// 	}
// }

func (f *RemoteStatusFetcher) fetch(uri string) (map[string]interface{}, error) {
	c, err := activitypub.NewClient(f.signAs.Actor.PublicKeyID(), f.signAs.PrivateKey)
	if err != nil {
		return nil, err
	}
	return c.Get(uri)
}

// FindOrCreate searches for a status by its URI. If the status is not found, it
// calls the given function to create a new status, stores that status in the
// database and returns it.
func (s *statuses) FindOrCreate(uri string, createFn func(string) (*models.Status, error)) (*models.Status, error) {
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
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Slice {
		var result []any
		for i := 0; i < val.Len(); i++ {
			result = append(result, val.Index(i).Interface())
		}
		return result
	}
	return nil
}
