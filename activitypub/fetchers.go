package activitypub

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/models"
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
	fmt.Println("RemoteActorFetcher.Fetch", uri)

	actor, err := FetchActor(f.db.Statement.Context, f.signAs, uri)
	if err != nil {
		return nil, err
	}

	published := actor.Published
	if published.IsZero() {
		published = time.Now()
	}

	u, err := url.Parse(actor.ID)
	if err != nil {
		return nil, err
	}

	return &models.Actor{
		ID:             snowflake.TimeToID(published),
		Type:           models.ActorType(actor.Type),
		Name:           actor.PreferredUsername,
		Domain:         u.Host,
		URI:            actor.ID,
		DisplayName:    actor.Name,
		Locked:         actor.ManuallyApprovesFollowers,
		Note:           actor.Summary,
		Avatar:         actor.Icon.URL,
		Header:         actor.Image.URL,
		InboxURL:       actor.Inbox,
		OutboxURL:      actor.Outbox,
		SharedInboxURL: actor.Endpoints.SharedInbox,
		PublicKey:      []byte(actor.PublicKey.PublicKeyPem),
		Attributes:     attachmentsToActorAttributes(actor.Attachments),
	}, nil
}

func attachmentsToActorAttributes(attachments []Attachment) []*models.ActorAttribute {
	return algorithms.Map(
		algorithms.Filter(
			attachments,
			propertyType("PropertyValue"),
		),
		objToActorAttribute,
	)
}

func objToActorAttribute(a Attachment) *models.ActorAttribute {
	return &models.ActorAttribute{
		Name:  a.Name,
		Value: a.Value,
	}
}

func propertyType(t string) func(Attachment) bool {
	return func(a Attachment) bool {
		return a.Type == t
	}
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
	fmt.Println("RemoteStatusFetcher.Fetch", uri)

	ctx, cancel := context.WithTimeout(f.db.Statement.Context, 5*time.Second)
	defer cancel()

	status, err := FetchStatus(ctx, f.signAs, uri)
	if err != nil {
		return nil, err
	}

	switch status.Type {
	case "Note", "Question":
		// cool
	default:
		return nil, fmt.Errorf("unsupported type %q", status.Type)
	}

	var visibility string
	for _, recipient := range anyToSlice(status.To) {
		switch recipient {
		case "https://www.w3.org/ns/activitystreams#Public":
			visibility = "public"
		case status.AttributedTo + "/followers":
			visibility = "limited"
		}
	}
	if visibility == "" {
		for _, recipient := range anyToSlice(status.CC) {
			switch recipient {
			case "https://www.w3.org/ns/activitystreams#Public":
				visibility = "public"
			case status.AttributedTo + "/followers":
				visibility = "limited"
			}
		}
	}
	if visibility == "" {
		return nil, fmt.Errorf("unsupported visibility %q: %v", visibility, status)
	}

	conv := &models.Conversation{
		Visibility: models.Visibility(visibility),
	}
	var inReplyTo *models.Status
	if status.InReplyTo != "" {
		inReplyTo, err = models.NewStatuses(f.db).FindOrCreate(status.InReplyTo, f.Fetch)
		if err != nil {
			return nil, err
		}
		conv = inReplyTo.Conversation
	}

	actors := NewRemoteActorFetcher(f.signAs, f.db)
	actor, err := models.NewActors(f.db).FindOrCreate(status.AttributedTo, actors.Fetch)
	if err != nil {
		return nil, err
	}

	st := &models.Status{
		ID:               snowflake.TimeToID(status.Published),
		UpdatedAt:        status.Updated,
		ActorID:          actor.ID,
		Actor:            actor,
		Conversation:     conv,
		InReplyToID:      inReplyToID(inReplyTo),
		InReplyToActorID: inReplyToActorID(inReplyTo),
		Sensitive:        status.Sensitive,
		SpoilerText:      status.Summary,
		Visibility:       conv.Visibility,
		URI:              status.ID,
		Note:             status.Content,
		Attachments:      attachmentsToStatusAttachments(status.Attachments),
	}

	for _, tag := range status.Tags {
		switch tag.Type {
		case "Mention":
			mention, err := models.NewActors(f.db).FindOrCreate(tag.Href, actors.Fetch)
			if err != nil {
				return nil, err
			}
			st.Mentions = append(st.Mentions, models.StatusMention{
				StatusID: st.ID,
				ActorID:  mention.ID,
				Actor:    mention,
			})
		case "Hashtag":
			st.Tags = append(st.Tags, models.StatusTag{
				StatusID: st.ID,
				Tag: &models.Tag{
					Name: strings.TrimLeft(tag.Name, "#"),
				},
			})
		}
	}

	if len(status.OneOf) > 0 {
		poll := &models.StatusPoll{
			StatusID:  st.ID,
			ExpiresAt: status.EndTime,
			Multiple:  false,
		}
		for _, option := range status.OneOf {
			if option.Type != "Note" {
				return nil, fmt.Errorf("invalid poll option type: %q", option.Type)
			}
			poll.Options = append(poll.Options, models.StatusPollOption{
				Title: option.Name,
				Count: option.Replies.TotalItems,
			})
		}
		st.Poll = poll
	}

	return st, nil
}

func attachmentsToStatusAttachments(attachments []any) []*models.StatusAttachment {
	return algorithms.Map(
		algorithms.Map(
			attachments,
			mapFromAny,
		),
		objToStatusAttachment,
	)
}
