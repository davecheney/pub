package activitypub

import (
	"crypto"
	"errors"
	"fmt"
	"net/http"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Inboxes struct {
	service *Service
	getKey  func(keyId string) (crypto.PublicKey, error)
}

func (i *Inboxes) Create(w http.ResponseWriter, r *http.Request) {
	// find the instance that this request is for.
	var instance models.Instance
	if err := i.service.db.Joins("Admin").Preload("Admin.Actor").First(&instance, "domain = ?", r.Host).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, err.Error(), http.StatusBadRequest) // TODO better error
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := i.validateSignature(r); err != nil {
		fmt.Println("validateSignature failed", err)
	}

	var body map[string]any
	if err := json.UnmarshalFull(r.Body, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// if we need to make an activity pub request, we need to sign it with the
	// instance's admin account.
	signAs := instance.Admin
	err := i.processActivity(signAs, body)
	if err != nil {
		fmt.Println("processActivity failed", stringFromAny(body["id"]), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// processActivity processes an activity. If the activity can be handled without
// blocking, it is handled immediately. If the activity requires blocking, it is
// queued for later processing.
func (i *Inboxes) processActivity(signAs *models.Account, body map[string]any) error {
	fmt.Println("processActivity: type:", stringFromAny(body["type"]), "id:", stringFromAny(body["id"]))
	typ := stringFromAny(body["type"])
	switch typ {
	case "Create":
		create := mapFromAny(body["object"])
		return i.processCreate(signAs, create)
	case "Announce":
		return i.processAnnounce(signAs, body)
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

func (i *Inboxes) processUndo(obj map[string]any) error {
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

func (i *Inboxes) processUndoAnnounce(obj map[string]any) error {
	id := stringFromAny(obj["id"])
	status, err := models.NewStatuses(i.service.db).FindByURI(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// already deleted
		return nil
	}
	if err != nil {
		return err
	}
	return i.service.db.Delete(status).Error
}

func (i *Inboxes) processUndoFollow(body map[string]any) error {
	actors := models.NewActors(i.service.db)
	actor, err := actors.FindByURI(stringFromAny(body["actor"]))
	if err != nil {
		return err
	}
	target, err := actors.FindByURI(stringFromAny(body["object"]))
	if err != nil {
		return err
	}
	relationships := models.NewRelationships(i.service.db)
	_, err = relationships.Unfollow(actor, target)
	return err
}

func (i *Inboxes) processAnnounce(signAs *models.Account, obj map[string]any) error {
	target := stringFromAny(obj["object"])
	statusFetcher := NewRemoteStatusFetcher(signAs, i.service.db)
	statuses := models.NewStatuses(i.service.db)
	original, err := statuses.FindOrCreate(target, statusFetcher.Fetch)
	if err != nil {
		return err
	}

	actorFetcher := NewRemoteActorFetcher(signAs, i.service.db)
	actors := models.NewActors(i.service.db)
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
	if err := i.service.db.Create(&conv).Error; err != nil {
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

	return i.service.db.Create(status).Error
}

func ptr[T any](v T) *T {
	return &v
}

func (i *Inboxes) processAdd(act map[string]any) error {
	target := stringFromAny(act["target"])
	switch target {
	case stringFromAny(act["actor"]) + "/collections/featured":
		status, err := models.NewStatuses(i.service.db).FindByURI(stringFromAny(act["object"]))
		if err != nil {
			return err
		}
		actor, err := models.NewActors(i.service.db).FindByURI(stringFromAny(act["actor"]))
		if err != nil {
			return err
		}
		if status.ActorID != actor.ID {
			return errors.New("actor is not the author of the status")
		}
		reactions := models.NewReactions(i.service.db)
		return reactions.Pin(status, actor)
	default:
		x, _ := marshalIndent(act)
		fmt.Println("processAdd:", string(x))
		return errors.New("not implemented")
	}
}

func (i *Inboxes) processRemove(act map[string]any) error {
	target := stringFromAny(act["target"])
	switch target {
	case stringFromAny(act["actor"]) + "/collections/featured":
		status, err := models.NewStatuses(i.service.db).FindByURI(stringFromAny(act["object"]))
		if err != nil {
			return err
		}
		actor, err := models.NewActors(i.service.db).FindByURI(stringFromAny(act["actor"]))
		if err != nil {
			return err
		}
		if status.ActorID != actor.ID {
			return errors.New("actor is not the author of the status")
		}
		reactions := models.NewReactions(i.service.db)
		return reactions.Unpin(status, actor)
	default:
		x, _ := marshalIndent(act)
		fmt.Println("processRemove:", string(x))
		return errors.New("not implemented")
	}
}

func (i *Inboxes) processCreate(signAs *models.Account, create map[string]any) error {
	typ := stringFromAny(create["type"])
	switch typ {
	case "Note":
		return i.processCreateNote(signAs, create)
	default:
		return fmt.Errorf("unknown create object type: %q", typ)
	}
}

func (i *Inboxes) processCreateNote(signAs *models.Account, create map[string]any) error {
	uri := stringFromAny(create["atomUri"])
	if uri == "" {
		return errors.New("missing atomUri")
	}

	_, err := models.NewStatuses(i.service.db).FindOrCreate(uri, func(string) (*models.Status, error) {
		fetcher := NewRemoteActorFetcher(signAs, i.service.db)
		actor, err := models.NewActors(i.service.db).FindOrCreate(stringFromAny(create["attributedTo"]), fetcher.Fetch)
		if err != nil {
			return nil, err
		}

		published, err := timeFromAny(create["published"])
		if err != nil {
			return nil, err
		}

		var inReplyTo *models.Status
		if inReplyToAtomUri, ok := create["inReplyTo"].(string); ok {
			remoteStatusFetcher := NewRemoteStatusFetcher(signAs, i.service.db)
			inReplyTo, err = models.NewStatuses(i.service.db).FindOrCreate(inReplyToAtomUri, remoteStatusFetcher.Fetch)
			if err != nil {
				fmt.Println("inReplyToAtomUri:", inReplyToAtomUri, "err:", err)
			}
		}

		vis := visiblity(create)
		var conversationID uint32
		if inReplyTo != nil {
			conversationID = inReplyTo.ConversationID
		} else {
			conv, err := models.NewConversations(i.service.db).New(vis)
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
		}
		for _, att := range anyToSlice(create["attachment"]) {
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

func (i *Inboxes) processAccept(obj map[string]any) error {
	typ := stringFromAny(obj["type"])
	switch typ {
	case "Follow":
		return i.processAcceptFollow(obj)
	default:
		return fmt.Errorf("unknown accept object type: %q", typ)
	}
}

func (i *Inboxes) processAcceptFollow(obj map[string]any) error {
	// consume
	return nil
}

func (i *Inboxes) processFollow(body map[string]any) error {
	actors := models.NewActors(i.service.db)
	actor, err := actors.FindByURI(stringFromAny(body["actor"]))
	if err != nil {
		return err
	}
	target, err := actors.FindByURI(stringFromAny(body["object"]))
	if err != nil {
		return err
	}
	relationships := models.NewRelationships(i.service.db)
	_, err = relationships.Follow(actor, target)
	return err
}

func (i *Inboxes) processUpdate(update map[string]any) error {
	id := stringFromAny(update["id"])
	var status models.Status
	if err := i.service.db.Where("uri = ?", id).First(&status).Error; err != nil {
		x, _ := marshalIndent(update)
		fmt.Println("processUpdate", string(x), err)
		return err
	}
	updated, err := timeFromAny(update["published"])
	if err != nil {
		return err
	}
	status.UpdatedAt = updated
	status.Note = stringFromAny(update["content"])
	return i.service.db.Save(&status).Error
}

func (i *Inboxes) processDelete(body map[string]any) error {
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

func (i *Inboxes) processDeleteStatus(uri string) error {
	// load status to delete it so we can fire the delete hooks.
	var status models.Status
	if err := i.service.db.Where("uri = ?", uri).First(&status).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// already deleted
			return nil
		}
		return err
	}
	return i.service.db.Delete(&status).Error
}

func (i *Inboxes) processDeleteActor(uri string) error {
	// load actor to delete it so we can fire the delete hooks.
	var actor models.Actor
	if err := i.service.db.Where("uri = ?", uri).First(&actor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// already deleted
			return nil
		}
		return err
	}
	return i.service.db.Delete(&actor).Error
}

func (i *Inboxes) validateSignature(r *http.Request) error {
	verifier, err := httpsig.NewVerifier(r)
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
	return ""
}
