package activitypub

import (
	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/models"
)

func attachmentsToActorAttributes(attachments []Attachment) []*models.ActorAttachment {
	return algorithms.Map(
		algorithms.Filter(
			attachments,
			propertyType("PropertyValue"),
		),
		objToActorAttribute,
	)
}

func objToActorAttribute(a Attachment) *models.ActorAttachment {
	return &models.ActorAttachment{
		Name:  a.Name,
		Value: a.Value,
	}
}

func propertyType(t string) func(Attachment) bool {
	return func(a Attachment) bool {
		return a.Type == t
	}
}

// type RemoteStatusFetcher struct {
// 	signAs *models.Account
// 	db     *gorm.DB
// }

// func NewRemoteStatusFetcher(signAs *models.Account, db *gorm.DB) *RemoteStatusFetcher {
// 	return &RemoteStatusFetcher{
// 		signAs: signAs,
// 		db:     db,
// 	}
// }

// func (f *RemoteStatusFetcher) Fetch(uri string) (*models.Status, error) {
// 	fmt.Println("RemoteStatusFetcher.Fetch", uri)

// 	ctx, cancel := context.WithTimeout(f.db.Statement.Context, 5*time.Second)
// 	defer cancel()

// 	c, err := NewClient(f.signAs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var status Status
// 	if err := c.Fetch(ctx, uri, &status); err != nil {
// 		return nil, err
// 	}

// 	switch status.Type {
// 	case "Note", "Question":
// 		// cool
// 	default:
// 		return nil, fmt.Errorf("unsupported type %q", status.Type)
// 	}

// 	var visibility string
// 	for _, recipient := range status.To {
// 		switch recipient {
// 		case "https://www.w3.org/ns/activitystreams#Public":
// 			visibility = "public"
// 		case status.AttributedTo + "/followers":
// 			visibility = "limited"
// 		}
// 	}
// 	if visibility == "" {
// 		for _, recipient := range status.CC {
// 			switch recipient {
// 			case "https://www.w3.org/ns/activitystreams#Public":
// 				visibility = "public"
// 			case status.AttributedTo + "/followers":
// 				visibility = "limited"
// 			}
// 		}
// 	}
// 	if visibility == "" {
// 		return nil, fmt.Errorf("unsupported visibility %q: %v", visibility, status)
// 	}

// 	conv := &models.Conversation{
// 		Visibility: models.Visibility(visibility),
// 	}
// 	var inReplyTo *models.Status
// 	if status.InReplyTo != "" {
// 		inReplyTo, err = models.NewStatuses(f.db).FindOrCreateByURI(status.InReplyTo)
// 		if err != nil {
// 			return nil, err
// 		}
// 		conv = inReplyTo.Conversation
// 	}

// 	actor, err := models.NewActors(f.db).FindOrCreateByURI(status.AttributedTo)
// 	if err != nil {
// 		return nil, err
// 	}

// 	st := &models.Status{
// 		ObjectID:         snowflake.TimeToID(status.Published),
// 		UpdatedAt:        status.Updated,
// 		ActorID:          actor.ObjectID,
// 		Actor:            actor,
// 		Conversation:     conv,
// 		InReplyToID:      inReplyToID(inReplyTo),
// 		InReplyToActorID: inReplyToActorID(inReplyTo),
// 		Visibility:       conv.Visibility,
// 	}

// for _, tag := range status.Tags {
// 	switch tag.Type {
// 	case "Mention":
// 		mention, err := models.NewActors(f.db).FindOrCreate(tag.Href, actors.Fetch)
// 		if err != nil {
// 			return nil, err
// 		}
// 		st.Mentions = append(st.Mentions, models.StatusMention{
// 			StatusID: st.ID,
// 			ActorID:  mention.ID,
// 			Actor:    mention,
// 		})
// 	case "Hashtag":
// 		st.Tags = append(st.Tags, models.StatusTag{
// 			StatusID: st.ID,
// 			Tag: &models.Tag{
// 				Name: strings.TrimLeft(tag.Name, "#"),
// 			},
// 		})
// 	}
// }

// if len(status.OneOf) > 0 {
// 	poll := &models.StatusPoll{
// 		StatusID:  st.ID,
// 		ExpiresAt: status.EndTime,
// 		Multiple:  false,
// 	}
// 	for _, option := range status.OneOf {
// 		if option.Type != "Note" {
// 			return nil, fmt.Errorf("invalid poll option type: %q", option.Type)
// 		}
// 		poll.Options = append(poll.Options, models.StatusPollOption{
// 			Title: option.Name,
// 			Count: option.Replies.TotalItems,
// 		})
// 	}
// 	st.Poll = poll
// }

// return st, nil
// }

// func attachmentsToStatusAttachments(attachments []any) []*models.StatusAttachment {
// 	return algorithms.Map(
// 		algorithms.Map(
// 			attachments,
// 			mapFromAny,
// 		),
// 		objToStatusAttachment,
// 	)
// }
