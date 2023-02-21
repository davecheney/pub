package activitypub

import (
	"crypto"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	var body map[string]any
	if err := json.UnmarshalFull(r.Body, &body); err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	// if we need to make an activity pub request, we need to sign it with the
	// instance's admin account.
	processor := &inboxProcessor{
		req:    r,
		db:     env.DB,
		signAs: instance.Admin,
		getKey: env.GetKey,
	}

	err := processor.processActivity(body)
	if err != nil {
		return fmt.Errorf("processActivity failed: %s: %w ", stringFromAny(body["id"]), err)
	}
	w.WriteHeader(http.StatusAccepted)
	return nil
}

type inboxProcessor struct {
	req    *http.Request
	db     *gorm.DB
	signAs *models.Account
	getKey func(keyID string) (crypto.PublicKey, error)
}

// processActivity processes an activity. If the activity can be handled without
// blocking, it is handled immediately. If the activity requires blocking, it is
// queued for later processing.
func (i *inboxProcessor) processActivity(body map[string]any) error {
	typ := stringFromAny(body["type"])
	id := stringFromAny(body["id"])
	fmt.Println("processActivity: type:", typ, "id:", id)
	if typ == "" {
		x, _ := marshalIndent(body)
		fmt.Println("processActivity: body:", string(x))
		return httpx.Error(http.StatusBadRequest, errors.New("missing type"))
	}
	switch typ {
	case "Delete":
		// Delete is a special case, as we may not have the actor in our database.
		// In that case, check the actor exists locally, and if it does, then
		// validate the signature.
		return i.processDelete(body)
	default:
		if err := i.validateSignature(); err != nil {
			return httpx.Error(http.StatusUnauthorized, err)
		}
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
	original, err := models.NewStatuses(i.db).FindOrCreate(target, statusFetcher.Fetch)
	if err != nil {
		return err
	}

	actorFetcher := NewRemoteActorFetcher(i.signAs, i.db)
	actor, err := models.NewActors(i.db).FindOrCreate(stringFromAny(obj["actor"]), actorFetcher.Fetch)
	if err != nil {
		return err
	}

	publishedAt, updatedAt, err := publishedAndUpdated(obj)
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
		ID:               snowflake.TimeToID(publishedAt),
		UpdatedAt:        updatedAt,
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
	switch obj := act["object"].(type) {
	case map[string]any:
		return i.processAddObject(act, obj)
	case string:
		return i.processAddPin(act)
	default:
		return fmt.Errorf("processAdd: unknown object type: %T", obj)
	}
}

func (i *inboxProcessor) processAddObject(act map[string]any, obj map[string]any) error {
	x, _ := marshalIndent(act)
	fmt.Println("processAddObject:", string(x))
	return errors.New("not implemented")
}

func (i *inboxProcessor) processAddPin(act map[string]any) error {
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
		_, err = models.NewReactions(i.db).Pin(status, actor)
		return err
	default:
		x, _ := marshalIndent(act)
		fmt.Println("processAddPin:", string(x))
		return errors.New("not implemented")
	}
}

func (i *inboxProcessor) processRemove(act map[string]any) error {
	switch obj := act["object"].(type) {
	case map[string]any:
		return i.processRemoveObject(act, obj)
	case string:
		return i.processRemovePin(act)
	default:
		return fmt.Errorf("processRemove: unknown type: %T", obj)
	}
}

func (i *inboxProcessor) processRemoveObject(act, obj map[string]any) error {
	x, _ := marshalIndent(act)
	fmt.Println("processRemoveObject:", string(x))
	return errors.New("not implemented")
}

func (i *inboxProcessor) processRemovePin(act map[string]any) error {
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
		_, err = reactions.Unpin(status, actor)
		return err
	default:
		x, _ := marshalIndent(act)
		fmt.Println("processRemovePin:", string(x))
		return errors.New("not implemented")
	}
}

func (i *inboxProcessor) processCreate(create map[string]any) error {
	typ := stringFromAny(create["type"])
	switch typ {
	case "Note", "Question":
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

		publishedAt, updatedAt, err := publishedAndUpdated(create)
		if err != nil {
			return nil, err
		}

		var inReplyTo *models.Status
		if inReplyToAtomUri, ok := create["inReplyTo"].(string); ok {
			remoteStatusFetcher := NewRemoteStatusFetcher(i.signAs, i.db)
			inReplyTo, err = models.NewStatuses(i.db).FindOrCreate(inReplyToAtomUri, remoteStatusFetcher.Fetch)
			if err != nil {
				if err := i.retry(uri, inReplyToAtomUri, err); err != nil {
					return nil, err
				}
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
			ID:               snowflake.TimeToID(publishedAt),
			UpdatedAt:        updatedAt,
			ActorID:          actor.ID,
			Actor:            actor,
			ConversationID:   conversationID,
			URI:              uri,
			InReplyToID:      inReplyToID(inReplyTo),
			InReplyToActorID: inReplyToActorID(inReplyTo),
			Sensitive:        boolFromAny(create["sensitive"]),
			SpoilerText:      stringFromAny(create["summary"]),
			Visibility:       models.Visibility(vis),
			Language:         "en",
			Note:             stringFromAny(create["content"]),
			Attachments:      attachmentsToStatusAttachments(anyToSlice(create["attachment"])),
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
					Actor:    mention,
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

		if _, ok := create["oneOf"]; ok {
			st.Poll, err = objToStatusPoll(create)
			if err != nil {
				return nil, err
			}
			st.Poll.StatusID = st.ID
		}

		return st, nil
	})
	if err != nil {
		b, _ := marshalIndent(create)
		fmt.Println("processCreate", string(b), err)
	}
	return err
}

// retry adds the uri and the parent to the retry queue.
func (i *inboxProcessor) retry(uri, parent string, err error) error {
	upsert := clause.OnConflict{
		UpdateAll: true,
	}
	if err := i.db.Clauses(upsert).Create(&models.ActivitypubRefresh{
		URI:         parent,
		Attempts:    1,
		LastAttempt: time.Now(),
		LastResult:  err.Error(),
	}).Error; err != nil {
		return err
	}
	if err := i.db.Clauses(upsert).Create(&models.ActivitypubRefresh{
		URI:       uri,
		DependsOn: parent,
	}).Error; err != nil {
		return err
	}
	return nil
}

// publishedAndUpdated returns the published and updated times for the given object.
// If the object does not have a published time, an error is returned.
// If the object does not have an updated time, updated at is set to published at.
func publishedAndUpdated(obj map[string]any) (time.Time, time.Time, error) {
	published, err := time.Parse(time.RFC3339, stringFromAny(obj["published"]))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	updated, err := time.Parse(time.RFC3339, stringFromAny(obj["updated"]))
	if err != nil {
		updated = published
	}
	return published, updated, nil
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

func objToStatusAttachment(obj map[string]any) *models.StatusAttachment {
	fmt.Println("objToStatusAttachment:", obj)
	return &models.StatusAttachment{
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
	case "Note", "Question":
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
	_, updatedAt, err := publishedAndUpdated(update)
	if err != nil {
		return err
	}

	status.UpdatedAt = updatedAt
	status.Note = stringFromAny(update["content"])
	if status.Poll != nil {
		if err := i.db.Delete(status.Poll).Error; err != nil {
			return err
		}
	}

	if _, ok := update["oneOf"]; ok {
		status.Poll, err = objToStatusPoll(update)
		if err != nil {
			return err
		}
		status.Poll.StatusID = status.ID
	}

	// TODO handle attachments
	return i.db.Save(&status).Error
}

func objToStatusPoll(obj map[string]any) (*models.StatusPoll, error) {
	expiresAt, err := timeFromAny(obj["endTime"])
	if err != nil {
		return nil, err
	}

	poll := &models.StatusPoll{
		ExpiresAt: expiresAt,
		Multiple:  false,
	}

	oneOf := anyToSlice(obj["oneOf"])
	for _, o := range oneOf {
		option := mapFromAny(o)
		if option["type"] != "Note" {
			return nil, fmt.Errorf("invalid poll option type: %q", option["type"])
		}

		poll.Options = append(poll.Options, models.StatusPollOption{
			Title: stringFromAny(option["name"]),
			Count: intFromAny(mapFromAny(option["replies"])["totalItems"]),
		})
	}

	return poll, nil
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
	actor.PublicKey = []byte(stringFromAny(mapFromAny(update["publicKey"])["publicKeyPem"]))

	// todo update attributes

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
	if err := i.validateSignature(); err != nil {
		return httpx.Error(http.StatusUnauthorized, err)
	}

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
	if err := i.validateSignature(); err != nil {
		return httpx.Error(http.StatusUnauthorized, err)
	}
	return i.db.Delete(&actor).Error
}

func (i *inboxProcessor) validateSignature() error {
	verifier, err := httpsig.NewVerifier(i.req)
	if err != nil {
		return err
	}
	pubKey, err := i.getKey(verifier.KeyId())
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
	return "direct" // hack
}
