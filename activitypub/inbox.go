package activitypub

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

func InboxCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	// find the instance that this request is for.
	var instance models.Instance
	if err := env.DB.Joins("Admin").Preload("Admin.Actor").Take(&instance, "domain = ?", r.Host).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}

	if err := validateSignature(env, r); err != nil {
		return httpx.Error(http.StatusUnauthorized, err)
	}

	var body map[string]any
	if err := json.UnmarshalFull(r.Body, &body); err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	// if we need to make an activity pub request, we need to sign it with the
	// instance's admin account.
	processor := &inboxProcessor{
		db:     env.DB,
		signAs: instance.Admin,
	}

	err := processor.processActivity(body)
	if err != nil {
		return fmt.Errorf("processActivity failed: %s: %w ", stringFromAny(body["id"]), err)
	}
	w.WriteHeader(http.StatusAccepted)
	return nil
}

type inboxProcessor struct {
	db     *gorm.DB
	signAs *models.Account
}

// processActivity processes an activity. If the activity can be handled without
// blocking, it is handled immediately. If the activity requires blocking, it is
// queued for later processing.
func (i *inboxProcessor) processActivity(body map[string]any) error {
	fmt.Println("processActivity: type:", stringFromAny(body["type"]), "id:", stringFromAny(body["id"]))
	typ := stringFromAny(body["type"])
	switch typ {
	case "Create":
		create := mapFromAny(body["object"])
		return i.processCreate(create)
	case "Announce":
		return i.processAnnounce(body)
	case "Undo":
		undo := mapFromAny(body["object"])
		return i.processUndo(undo)
	case "Update":
		update := mapFromAny(body["object"])
		return i.processUpdate(update)
	case "Delete":
		return i.processDelete(body)
	case "Follow":
		return i.processFollow(body)
	case "Accept":
		accept := mapFromAny(body["object"])
		return i.processAccept(accept)
	case "Add":
		return i.processAdd(body)
	case "Remove":
		return i.processRemove(body)
	default:
		return errors.New("unknown activity type " + typ)
	}
}

func (i *inboxProcessor) processUndo(obj map[string]any) error {
	typ := stringFromAny(obj["type"])
	switch typ {
	case "Announce":
		return i.processUndoAnnounce(obj)
	case "Follow":
		return i.processUndoFollow(obj)
	default:
		return fmt.Errorf("unknown undo object type: %q", typ)
	}
}

func (i *inboxProcessor) processUndoAnnounce(obj map[string]any) error {
	id := stringFromAny(obj["id"])
	status, err := models.NewStatuses(i.db).FindByURI(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// already deleted
		return nil
	}
	if err != nil {
		return err
	}
	return i.db.Delete(status).Error
}

func (i *inboxProcessor) processUndoFollow(body map[string]any) error {
	actors := models.NewActors(i.db)
	actor, err := actors.FindByURI(stringFromAny(body["actor"]))
	if err != nil {
		return err
	}
	target, err := actors.FindByURI(stringFromAny(body["object"]))
	if err != nil {
		return err
	}
	relationships := models.NewRelationships(i.db)
	_, err = relationships.Unfollow(actor, target)
	return err
}

func (i *inboxProcessor) processAnnounce(obj map[string]any) error {
	target := stringFromAny(obj["object"])
	statusFetcher := NewRemoteStatusFetcher(i.signAs, i.db)
	statuses := models.NewStatuses(i.db)
	original, err := statuses.FindOrCreate(target, statusFetcher.Fetch)
	if err != nil {
		return err
	}

	actorFetcher := NewRemoteActorFetcher(i.signAs, i.db)
	actors := models.NewActors(i.db)
	actor, err := actors.FindOrCreate(stringFromAny(obj["actor"]), actorFetcher.Fetch)
	if err != nil {
		return err
	}

	published, err := timeFromAny(obj["published"])
	if err != nil {
		return err
	}

	conv := models.Conversation{
		Visibility: "public",
	}
	if err := i.db.Create(&conv).Error; err != nil {
		return err
	}

	status := &models.Status{
		ID:               snowflake.TimeToID(published),
		ActorID:          actor.ID,
		Actor:            actor,
		ConversationID:   conv.ID,
		URI:              stringFromAny(obj["id"]),
		InReplyToID:      nil,
		InReplyToActorID: nil,
		Sensitive:        false,
		SpoilerText:      "",
		Visibility:       "public",
		Language:         "",
		Note:             "",
		ReblogID:         &original.ID,
	}

	return i.db.Create(status).Error
}

func (i *inboxProcessor) processAdd(act map[string]any) error {
	target := stringFromAny(act["target"])
	switch target {
	case stringFromAny(act["actor"]) + "/collections/featured":
		status, err := models.NewStatuses(i.db).FindByURI(stringFromAny(act["object"]))
		if err != nil {
			return err
		}
		actor, err := models.NewActors(i.db).FindByURI(stringFromAny(act["actor"]))
		if err != nil {
			return err
		}
		if status.ActorID != actor.ID {
			return errors.New("actor is not the author of the status")
		}
		reactions := models.NewReactions(i.db)
		return reactions.Pin(status, actor)
	default:
		x, _ := marshalIndent(act)
		fmt.Println("processAdd:", string(x))
		return errors.New("not implemented")
	}
}

func (i *inboxProcessor) processRemove(act map[string]any) error {
	target := stringFromAny(act["target"])
	switch target {
	case stringFromAny(act["actor"]) + "/collections/featured":
		status, err := models.NewStatuses(i.db).FindByURI(stringFromAny(act["object"]))
		if err != nil {
			return err
		}
		actor, err := models.NewActors(i.db).FindByURI(stringFromAny(act["actor"]))
		if err != nil {
			return err
		}
		if status.ActorID != actor.ID {
			return errors.New("actor is not the author of the status")
		}
		reactions := models.NewReactions(i.db)
		return reactions.Unpin(status, actor)
	default:
		x, _ := marshalIndent(act)
		fmt.Println("processRemove:", string(x))
		return errors.New("not implemented")
	}
}

func (i *inboxProcessor) processCreate(create map[string]any) error {
	typ := stringFromAny(create["type"])
	switch typ {
	case "Note":
		return i.processCreateNote(create)
	default:
		return fmt.Errorf("unknown create object type: %q", typ)
	}
}

func (i *inboxProcessor) processCreateNote(create map[string]any) error {
	uri := stringFromAny(create["atomUri"])
	if uri == "" {
		return errors.New("missing atomUri")
	}

	_, err := models.NewStatuses(i.db).FindOrCreate(uri, func(string) (*models.Status, error) {
		fetcher := NewRemoteActorFetcher(i.signAs, i.db)
		actor, err := models.NewActors(i.db).FindOrCreate(stringFromAny(create["attributedTo"]), fetcher.Fetch)
		if err != nil {
			return nil, err
		}

		published, err := timeFromAny(create["published"])
		if err != nil {
			return nil, err
		}

		var inReplyTo *models.Status
		if inReplyToAtomUri, ok := create["inReplyTo"].(string); ok {
			remoteStatusFetcher := NewRemoteStatusFetcher(i.signAs, i.db)
			inReplyTo, err = models.NewStatuses(i.db).FindOrCreate(inReplyToAtomUri, remoteStatusFetcher.Fetch)
			if err != nil {
				fmt.Println("inReplyToAtomUri:", inReplyToAtomUri, "err:", err)
			}
		}

		vis := visiblity(create)
		var conversationID uint32
		if inReplyTo != nil {
			conversationID = inReplyTo.ConversationID
		} else {
			conv, err := models.NewConversations(i.db).New(vis)
			if err != nil {
				return nil, err
			}
			conversationID = conv.ID
		}

		st := &models.Status{
			ID:               snowflake.TimeToID(published),
			ActorID:          actor.ID,
			ConversationID:   conversationID,
			URI:              uri,
			InReplyToID:      inReplyToID(inReplyTo),
			InReplyToActorID: inReplyToActorID(inReplyTo),
			Sensitive:        boolFromAny(create["sensitive"]),
			SpoilerText:      stringFromAny(create["summary"]),
			Visibility:       vis,
			Language:         "en",
			Note:             stringFromAny(create["content"]),
			Attachments:      algorithms.Map(algorithms.Map(anyToSlice(create["attachment"]), mapFromAny), objToStatusAttachment),
		}
		// and here
		for _, tag := range anyToSlice(create["tag"]) {
			t := mapFromAny(tag)
			switch t["type"] {
			case "Mention":
				mention, err := models.NewActors(i.db).FindOrCreate(stringFromAny(t["href"]), fetcher.Fetch)
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
	})
	if err != nil {
		b, _ := marshalIndent(create)
		fmt.Println("processCreate", string(b), err)
	}
	return err
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

func objToStatusAttachment(obj map[string]any) models.StatusAttachment {
	fmt.Println("objToStatusAttachment:", obj)
	return models.StatusAttachment{
		Attachment: models.Attachment{
			ID:        snowflake.Now(),
			MediaType: stringFromAny(obj["mediaType"]),
			URL:       stringFromAny(obj["url"]),
			Name:      stringFromAny(obj["name"]),
			Width:     intFromAny(obj["width"]),
			Height:    intFromAny(obj["height"]),
			Blurhash:  stringFromAny(obj["blurhash"]),
		},
	}
}

func (i *inboxProcessor) processAccept(obj map[string]any) error {
	typ := stringFromAny(obj["type"])
	switch typ {
	case "Follow":
		return i.processAcceptFollow(obj)
	default:
		return fmt.Errorf("unknown accept object type: %q", typ)
	}
}

func (i *inboxProcessor) processAcceptFollow(obj map[string]any) error {
	// consume
	return nil
}

func (i *inboxProcessor) processFollow(body map[string]any) error {
	actors := models.NewActors(i.db)
	actor, err := actors.FindByURI(stringFromAny(body["actor"]))
	if err != nil {
		return err
	}
	target, err := actors.FindByURI(stringFromAny(body["object"]))
	if err != nil {
		return err
	}
	relationships := models.NewRelationships(i.db)
	_, err = relationships.Follow(actor, target)
	return err
}

func (i *inboxProcessor) processUpdate(update map[string]any) error {
	typ := stringFromAny(update["type"])
	switch typ {
	case "Note":
		return i.processUpdateStatus(update)
	case "Person":
		return i.processUpdateActor(update)
	default:
		return fmt.Errorf("unknown update object type: %q", typ)
	}
}

func (i *inboxProcessor) processUpdateStatus(update map[string]any) error {
	id := stringFromAny(update["id"])
	statusFetcher := NewRemoteStatusFetcher(i.signAs, i.db)
	status, err := models.NewStatuses(i.db).FindOrCreate(id, statusFetcher.Fetch)
	if err != nil {
		return err
	}
	updated, err := timeFromAny(update["published"])
	if err != nil {
		return err
	}
	status.UpdatedAt = updated
	status.Note = stringFromAny(update["content"])

	// TODO handle polls and attachments
	return i.db.Save(&status).Error
}

func (i *inboxProcessor) processUpdateActor(update map[string]any) error {
	id := stringFromAny(update["id"])
	actorFetcher := NewRemoteActorFetcher(i.signAs, i.db)
	actor, err := models.NewActors(i.db).FindOrCreate(id, actorFetcher.Fetch)
	if err != nil {
		return err
	}
	actor.Name = stringFromAny(update["preferredUsername"])
	actor.DisplayName = stringFromAny(update["name"])
	actor.Locked = boolFromAny(update["manuallyApprovesFollowers"])
	actor.Note = stringFromAny(update["summary"])
	actor.Avatar = stringFromAny(mapFromAny(update["icon"])["url"])
	actor.Header = stringFromAny(mapFromAny(update["image"])["url"])
	actor.Attachments = anyToSlice(update["attachment"])
	actor.PublicKey = []byte(stringFromAny(mapFromAny(update["publicKey"])["publicKeyPem"]))

	return i.db.Save(&actor).Error
}

func (i *inboxProcessor) processDelete(body map[string]any) error {
	obj := body["object"]
	switch obj := obj.(type) {
	case map[string]any:
		return i.processDeleteStatus(stringFromAny(obj["id"]))
	case string:
		return i.processDeleteActor(obj)
	default:
		typ := stringFromAny(body["type"])
		x, _ := marshalIndent(body)
		return fmt.Errorf("unknown delete object type: %q: %s", typ, string(x))
	}
}

func (i *inboxProcessor) processDeleteStatus(uri string) error {
	// load status to delete it so we can fire the delete hooks.
	status, err := models.NewStatuses(i.db).FindByURI(uri)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// already deleted
			return nil
		}
		return err
	}
	return i.db.Delete(&status).Error
}

func (i *inboxProcessor) processDeleteActor(uri string) error {
	// load actor to delete it so we can fire the delete hooks.
	actor, err := models.NewActors(i.db).FindByURI(uri)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// already deleted
			return nil
		}
		return err
	}
	return i.db.Delete(&actor).Error
}

func validateSignature(env *Env, r *http.Request) error {
	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		return err
	}
	pubKey, err := env.GetKey(verifier.KeyId())
	if err != nil {
		return err
	}
	if err := verifier.Verify(pubKey, httpsig.RSA_SHA256); err != nil {
		return err
	}
	return nil
}

func visiblity(obj map[string]any) string {
	actor := stringFromAny(obj["attributedTo"])
	for _, recipient := range anyToSlice(obj["to"]) {
		switch recipient {
		case "https://www.w3.org/ns/activitystreams#Public":
			return "public"
		case actor + "/followers":
			return "limited"
		}
	}
	for _, recipient := range anyToSlice(obj["cc"]) {
		switch recipient {
		case "https://www.w3.org/ns/activitystreams#Public":
			return "public"
		}
	}
	return ""
}
