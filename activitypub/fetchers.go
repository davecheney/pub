package activitypub

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/davecheney/pub/internal/activitypub"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
)

type RemoteActorFetcher struct {
	// signAs is the account that will be used to sign the request
	signAs *models.Account
	db     *gorm.DB
}

func NewRemoteActorFetcher(signAs *models.Account, db *gorm.DB) *RemoteActorFetcher {
	return &RemoteActorFetcher{
		signAs: signAs,
		db:     db,
	}
}

func (f *RemoteActorFetcher) Fetch(uri string) (*models.Actor, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	obj, err := f.fetch(uri)
	if err != nil {
		return nil, err
	}

	published := timeFromAnyOrZero(obj["published"])
	if published.IsZero() {
		published = time.Now()
	}

	return &models.Actor{
		ID:           snowflake.TimeToID(published),
		Type:         stringFromAny(obj["type"]),
		Name:         stringFromAny(obj["preferredUsername"]),
		Domain:       u.Host,
		URI:          stringFromAny(obj["id"]),
		DisplayName:  stringFromAny(obj["name"]),
		Locked:       boolFromAny(obj["manuallyApprovesFollowers"]),
		Note:         stringFromAny(obj["summary"]),
		Avatar:       stringFromAny(mapFromAny(obj["icon"])["url"]),
		Header:       stringFromAny(mapFromAny(obj["image"])["url"]),
		LastStatusAt: time.Now(),
		Attachments:  anyToSlice(obj["attachment"]),
		PublicKey:    []byte(stringFromAny(mapFromAny(obj["publicKey"])["publicKeyPem"])),
	}, nil
}

func (f *RemoteActorFetcher) fetch(uri string) (map[string]any, error) {
	fmt.Println("RemoteActorFetcher.fetch:", uri)
	c, err := activitypub.NewClient(f.db.Statement.Context, f.signAs)
	if err != nil {
		return nil, err
	}
	return c.Get(uri)
}

type RemoteStatusFetcher struct {
	signAs *models.Account
	db     *gorm.DB
}

func NewRemoteStatusFetcher(signAs *models.Account, db *gorm.DB) *RemoteStatusFetcher {
	return &RemoteStatusFetcher{
		signAs: signAs,
		db:     db,
	}
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
		x, _ := marshalIndent(obj)
		return nil, fmt.Errorf("unsupported visibility %q: %s", visibility, x)
	}

	var inReplyTo *models.Status
	if inReplyToURI := stringFromAny(obj["inReplyTo"]); inReplyToURI != "" {
		inReplyTo, err = models.NewStatuses(f.db).FindOrCreate(inReplyToURI, f.Fetch)
		if err != nil {
			fmt.Println("inReplyToURI", inReplyToURI, err)
		}
	}

	var conversationID uint32
	if inReplyTo != nil {
		conversationID = inReplyTo.ConversationID
	} else {
		conv := models.Conversation{
			Visibility: visibility,
		}
		if err := f.db.Create(&conv).Error; err != nil {
			return nil, err
		}
		conversationID = conv.ID
	}
	fetcher := NewRemoteActorFetcher(f.signAs, f.db)
	actor, err := models.NewActors(f.db).FindOrCreate(stringFromAny(obj["attributedTo"]), fetcher.Fetch)
	if err != nil {
		return nil, err
	}
	createdAt := timeFromAnyOrZero(obj["published"])

	st := &models.Status{
		ID:               snowflake.TimeToID(createdAt),
		ActorID:          actor.ID,
		Actor:            actor,
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
	for _, tag := range anyToSlice(obj["tag"]) {
		t := mapFromAny(tag)
		switch t["type"] {
		case "Mention":
			mention, err := models.NewActors(f.db).FindOrCreate(stringFromAny(t["href"]), fetcher.Fetch)
			if err != nil {
				return nil, err
			}
			st.Mentions = append(st.Mentions, models.StatusMention{
				StatusID: st.ID,
				ActorID:  mention.ID,
			})
		case "Hashtag":
			st.Tags = append(st.Tags, models.StatusTag{
				StatusID: st.ID,
				Tag: &models.Tag{
					Name: strings.TrimLeft(stringFromAny(t["name"]), "#"),
				},
			})
		}
	}
	return st, nil
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
	fmt.Println("RemoteStatusFetcher.fetch:", uri)
	c, err := activitypub.NewClient(f.db.Statement.Context, f.signAs)
	if err != nil {
		return nil, err
	}
	return c.Get(uri)
}
