package activitypub

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/models"
	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewInbox(db *gorm.DB) *InboxController {
	return &InboxController{}
}

type InboxController struct {
}

func (i *InboxController) Create(env *Env, w http.ResponseWriter, r *http.Request) error {
	var act Activity
	if err := json.UnmarshalFull(r.Body, &act); err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	// if we need to make an activity pub request, we need to sign it with the
	// instance's admin account.
	processor := &inboxProcessor{
		logger: env.Logger.With("instance", env.Instance.Domain),
		req:    r,
		db:     env.DB,
		signAs: env.Instance.Admin,
	}

	if err := processor.processActivity(&act); err != nil {
		return fmt.Errorf("processActivity failed: %s: %w ", act.ID, err)
	}
	w.WriteHeader(http.StatusAccepted)
	return nil
}

type inboxProcessor struct {
	logger *slog.Logger
	req    *http.Request
	db     *gorm.DB
	signAs *models.Account
}

// processActivity processes an activity. If the activity can be handled without
// blocking, it is handled immediately. If the activity requires blocking, it is
// queued for later processing.
func (i *inboxProcessor) processActivity(act *Activity) error {
	i.logger = i.logger.With("id", act.ID, "type", act.Type)
	i.logger.Info("processActivity")
	switch act.Type {
	case "":
		return httpx.Error(http.StatusBadRequest, errors.New("missing type"))
	case "Delete":
		// Delete is a special case, as we may not have the actor in our database.
		// In that case, check the actor exists locally, and if it does, then
		// validate the signature.
		// return i.processDelete(act)
		return nil
	default:
		if err := i.validateSignature(); err != nil {
			return httpx.Error(http.StatusUnauthorized, err)
		}
		switch act.Type {
		case "Create":
			create := mapFromAny(act.Object)
			return i.processCreate(create)
		case "Announce":
			return i.processAnnounce(act)
		case "Undo":
			undo := mapFromAny(act.Object)
			return i.processUndo(undo)
		case "Update":
			update := mapFromAny(act.Object)
			return i.processUpdate(update)
		case "Follow":
			return i.processFollow(act)
		case "Accept":
			accept := mapFromAny(act.Object)
			return i.processAccept(accept)
		case "Add":
			return i.processAdd(act)
		case "Remove":
			return i.processRemove(act)
		default:
			return errors.New("unknown activity type: " + act.Type)
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

func (i *inboxProcessor) processAnnounce(act *Activity) error {
	target := stringFromAny(act.Object)
	original, err := models.NewStatuses(i.db).FindOrCreateByURI(target)
	if err != nil {
		return err
	}
	actor, err := models.NewActors(i.db).FindOrCreateByURI(stringFromAny(act.Actor))
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := json.MarshalFull(&buf, act); err != nil {
		return err
	}

	var props map[string]any
	if err := json.UnmarshalFull(&buf, &props); err != nil {
		return err
	}
	obj := &models.Object{
		Properties: props,
	}
	if err := i.db.Create(&obj).Error; err != nil {
		return err
	}

	updatedAt := act.Updated
	if updatedAt.IsZero() {
		updatedAt = obj.ID.ToTime()
	}

	status := &models.Status{
		ObjectID:  obj.ID,
		UpdatedAt: updatedAt,
		ActorID:   actor.ObjectID,
		Actor:     actor,
		Conversation: &models.Conversation{
			Visibility: "public",
		},
		Visibility: "public",
		ReblogID:   &original.ObjectID,
	}

	return i.db.Save(status).Error
}

func (i *inboxProcessor) processAdd(act *Activity) error {
	switch obj := act.Object.(type) {
	case string:
		return i.processAddPin(act)
	default:
		return fmt.Errorf("processAdd: unknown type: %T", obj)
	}
}

func (i *inboxProcessor) processAddPin(act *Activity) error {
	actor := stringFromAny(act.Actor)
	switch act.Target {
	case actor + "/collections/featured":
		status, err := models.NewStatuses(i.db).FindByURI(stringFromAny(act.Object))
		if err != nil {
			return err
		}
		actor, err := models.NewActors(i.db).FindByURI(actor)
		if err != nil {
			return err
		}
		if status.ActorID != actor.ObjectID {
			return errors.New("actor is not the author of the status")
		}
		_, err = models.NewReactions(i.db).Pin(status, actor)
		return err
	default:
		return errors.New("add pin: unknown target: " + act.Target)
	}
}

func (i *inboxProcessor) processRemove(act *Activity) error {
	switch obj := act.Object.(type) {
	case string:
		return i.processRemovePin(act)
	default:
		return fmt.Errorf("processRemove: unknown type: %T", obj)
	}
}

func (i *inboxProcessor) processRemovePin(act *Activity) error {
	actor := stringFromAny(act.Actor)
	switch act.Target {
	case actor + "/collections/featured":
		status, err := models.NewStatuses(i.db).FindByURI(stringFromAny(act.Object))
		if err != nil {
			return err
		}
		actor, err := models.NewActors(i.db).FindByURI(actor)
		if err != nil {
			return err
		}
		if status.ActorID != actor.ObjectID {
			return errors.New("actor is not the author of the status")
		}
		reactions := models.NewReactions(i.db)
		_, err = reactions.Unpin(status, actor)
		return err
	default:
		return errors.New("remove pin: unknown target: " + act.Target)
	}
}

func (i *inboxProcessor) processCreate(create map[string]any) error {
	var obj = &models.Object{
		Properties: create,
	}
	if err := i.db.Create(obj).Error; err != nil {
		return err
	}
	return nil

	// typ := stringFromAny(create["type"])
	// switch typ {
	// case "Note", "Question":
	// 	return i.processCreateNote(create)
	// default:
	// 	return fmt.Errorf("unknown create object type: %q", typ)
	// }
}

func (i *inboxProcessor) processCreateNote(create map[string]any) error {
	uri := stringFromAny(create["atomUri"])
	_, err := models.NewStatuses(i.db).FindByURI(uri)
	switch err {
	case nil:
		// we already have this status
		return nil
	case gorm.ErrRecordNotFound:
		// we don't have this status
		actor, err := models.NewActors(i.db).FindOrCreateByURI(stringFromAny(create["attributedTo"]))
		if err != nil {
			return err
		}

		publishedAt, updatedAt, err := publishedAndUpdated(create)
		if err != nil {
			return err
		}

		conv := &models.Conversation{
			Visibility: models.Visibility(visiblity(create)),
		}
		var inReplyTo *models.Status
		if inReplyToAtomUri, ok := create["inReplyTo"].(string); ok {
			inReplyTo, err = models.NewStatuses(i.db).FindOrCreateByURI(inReplyToAtomUri)
			if err != nil {
				return err
			}
			conv = inReplyTo.Conversation
		}

		status := models.Status{
			ObjectID:         snowflake.TimeToID(publishedAt),
			UpdatedAt:        updatedAt,
			ActorID:          actor.ObjectID,
			Actor:            actor,
			Conversation:     conv,
			InReplyToID:      inReplyToID(inReplyTo),
			InReplyToActorID: inReplyToActorID(inReplyTo),
			Visibility:       conv.Visibility,
		}
		// for _, tag := range anyToSlice(create["tag"]) {
		// 	t := mapFromAny(tag)
		// 	switch t["type"] {
		// 	case "Mention":
		// 		mention, err := models.NewActors(i.db).FindOrCreate(stringFromAny(t["href"]), actors.Fetch)
		// 		if err != nil {
		// 			return err
		// 		}
		// 		status.Mentions = append(status.Mentions, models.StatusMention{
		// 			StatusID: status.ID,
		// 			ActorID:  mention.ID,
		// 			Actor:    mention,
		// 		})
		// 	case "Hashtag":
		// 		status.Tags = append(status.Tags, models.StatusTag{
		// 			StatusID: status.ID,
		// 			Tag: &models.Tag{
		// 				Name: strings.TrimLeft(stringFromAny(t["name"]), "#"),
		// 			},
		// 		})
		// 	}
		// }

		// if _, ok := create["oneOf"]; ok {
		// 	status.Poll, err = objToStatusPoll(create)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	status.Poll.StatusID = status.ID
		// }
		return i.db.Create(&status).Error
	default:
		// something else happened
		return err
	}
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
		return &inReplyTo.ObjectID
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
	return &models.StatusAttachment{
		Attachment: models.Attachment{
			ID:         snowflake.Now(),
			MediaType:  stringFromAny(obj["mediaType"]),
			URL:        stringFromAny(obj["url"]),
			Name:       stringFromAny(obj["name"]),
			Width:      intFromAny(obj["width"]),
			Height:     intFromAny(obj["height"]),
			Blurhash:   stringFromAny(obj["blurhash"]),
			FocalPoint: focalPoint(obj),
		},
	}
}

func focalPoint(obj map[string]any) models.FocalPoint {
	focalPoint := anyToSlice(obj["focalPoint"])
	var x, y float64
	if len(focalPoint) == 2 {
		x, _ = focalPoint[0].(float64)
		y, _ = focalPoint[1].(float64)
	}
	return models.FocalPoint{
		X: x,
		Y: y,
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

func (i *inboxProcessor) processFollow(act *Activity) error {
	actors := models.NewActors(i.db)
	actor, err := actors.FindByURI(stringFromAny(act.Actor))
	if err != nil {
		return err
	}
	target, err := actors.FindByURI(stringFromAny(act.Object))
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
	status, err := models.NewStatuses(i.db).FindOrCreateByURI(id)
	if err != nil {
		return err
	}
	_, updatedAt, err := publishedAndUpdated(update)
	if err != nil {
		return err
	}

	status.UpdatedAt = updatedAt
	// status.Note = stringFromAny(update["content"])
	// if status.Poll != nil {
	// 	if err := i.db.Delete(status.Poll).Error; err != nil {
	// 		return err
	// 	}
	// }

	// if _, ok := update["oneOf"]; ok {
	// 	status.Poll, err = objToStatusPoll(update)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	status.Poll.StatusID = status.ID
	// }

	// TODO handle attachments
	upsert := clause.OnConflict{
		UpdateAll: true,
	}
	return i.db.Clauses(upsert).Save(&status).Error
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
	actor, err := models.NewActors(i.db).FindOrCreateByURI(id)
	if err != nil {
		return err
	}
	actor.Name = stringFromAny(update["preferredUsername"])

	// todo update attributes

	return i.db.Save(&actor).Error
}

func (i *inboxProcessor) processDelete(act *Activity) error {
	switch obj := act.Object.(type) {
	case map[string]any:
		return i.processDeleteStatus(stringFromAny(obj["id"]))
	case string:
		return i.processDeleteActor(obj)
	default:
		return fmt.Errorf("unknown delete object type: %q: %v", obj, act)
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
	// use this form to avoid firing gorm's ErrNotFound mechanism;
	// most deletes we don't know about, and that's fine.
	var actors []*models.Actor
	if err := i.db.Limit(1).Where("uri = ?", uri).Find(&actors).Error; err != nil {
		return err
	}
	if len(actors) == 0 {
		// already deleted
		return nil
	}
	actor := actors[0]
	if err := i.validateSignature(); err != nil {
		return httpx.Error(http.StatusUnauthorized, err)
	}
	return i.db.Delete(actor).Error
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

func (i *inboxProcessor) getKey(keyID string) (crypto.PublicKey, error) {
	actor, err := models.NewActors(i.db).FindOrCreateByURI(trimKeyId(keyID))
	if err != nil {
		return nil, err
	}
	return pemToPublicKey(actor.PublicKey())
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

func pemToPublicKey(key []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("pemToPublicKey: invalid pem type: %s", block.Type)
	}
	var publicKey interface{}
	var err error
	if publicKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("pemToPublicKey: parsepkixpublickey: %w", err)
	}
	return publicKey, nil
}

// trimKeyId removes the #main-key suffix from the key id.
func trimKeyId(id string) string {
	if i := strings.Index(id, "#"); i != -1 {
		return id[:i]
	}
	return id
}
