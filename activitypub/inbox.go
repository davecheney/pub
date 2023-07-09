package activitypub

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davecheney/pub/internal/activitypub"
	"github.com/davecheney/pub/internal/httpx"
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
	var act map[string]any
	if err := json.UnmarshalFull(r.Body, &act); err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}
	id, ok := act["id"].(string)
	if !ok {
		return errors.New("missing id")
	}

	// if we need to make an activity pub request, we need to sign it with the
	// instance's admin account.
	processor := &inboxProcessor{
		logger: env.Logger.With("instance", env.Instance.Domain),
		req:    r,
		db:     env.DB,
		client: env.Client,
	}

	if err := processor.processActivity(act); err != nil {
		return fmt.Errorf("processActivity failed: %s: %w ", id, err)
	}
	w.WriteHeader(http.StatusAccepted)
	return nil
}

type inboxProcessor struct {
	logger *slog.Logger
	req    *http.Request
	db     *gorm.DB
	client *activitypub.Client
}

// processActivity processes an activity. If the activity can be handled without
// blocking, it is handled immediately. If the activity requires blocking, it is
// queued for later processing.
func (i *inboxProcessor) processActivity(act map[string]any) error {
	typ, ok := act["type"].(string)
	if !ok {
		return errors.New("missing type")
	}
	i.logger = i.logger.With("id", stringFromAny(act["id"]), "type", typ)
	i.logger.Info("processActivity")
	switch typ {
	case "":
		return httpx.Error(http.StatusBadRequest, errors.New("missing type"))
	case "Delete":
		// Delete is a special case, as we may not have the actor in our database.
		// In that case, check the actor exists locally, and if it does, then
		// validate the signature.
		return i.processDelete(act)
	default:
		if err := i.validateSignature(); err != nil {
			return httpx.Error(http.StatusUnauthorized, err)
		}
		switch typ {
		case "Create":
			create, ok := act["object"].(map[string]any)
			if !ok {
				return errors.New("create: missing object")
			}
			return i.processCreate(create)
		case "Announce":
			return i.processAnnounce(act)
		case "Undo":
			undo, ok := act["object"].(map[string]any)
			if !ok {
				return errors.New("undo: missing object")
			}
			return i.processUndo(undo)
		case "Update":
			update, ok := act["object"].(map[string]any)
			if !ok {
				return errors.New("update: missing object")
			}
			return i.processUpdate(update)
		case "Follow":
			return i.processFollow(act)
		case "Accept":
			accept, ok := act["object"].(map[string]any)
			if !ok {
				return errors.New("accept: missing object")
			}
			return i.processAccept(accept)
		case "Add":
			return i.processAdd(act)
		case "Remove":
			return i.processRemove(act)
		default:
			return errors.New("unknown activity type: " + typ)
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
	var target models.Object
	if err := i.db.Where("uri = ?", stringFromAny(obj["id"])).Take(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// already deleted
			return nil
		}
		return err
	}
	return i.db.Delete(&target).Error
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

func (i *inboxProcessor) processAnnounce(act map[string]any) error {
	return i.createObject(act)
}

func (i *inboxProcessor) processAdd(act map[string]any) error {
	obj, ok := act["object"]
	if !ok {
		return errors.New("add: missing object")
	}
	switch obj := obj.(type) {
	case string:
		return i.processAddPin(act)
	default:
		return fmt.Errorf("processAdd: unknown type: %T", obj)
	}
}

func (i *inboxProcessor) processAddPin(act map[string]any) error {
	actor, ok := act["actor"].(string)
	if !ok {
		return errors.New("add pin: missing actor")
	}
	target, ok := act["target"].(string)
	if !ok {
		return errors.New("add pin: missing target")
	}
	switch target {
	case actor + "/collections/featured":
		object, ok := act["object"].(string)
		if !ok {
			return errors.New("add pin: missing object")
		}
		status, err := models.NewStatuses(i.db).FindByURI(object)
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
		return errors.New("add pin: unknown target: " + target)
	}
}

func (i *inboxProcessor) processRemove(act map[string]any) error {
	obj, ok := act["object"]
	if !ok {
		return errors.New("remove: missing object")
	}
	switch obj := obj.(type) {
	case string:
		return i.processRemovePin(act)
	default:
		return fmt.Errorf("processRemove: unknown type: %T", obj)
	}
}

func (i *inboxProcessor) processRemovePin(act map[string]any) error {
	actor, ok := act["actor"].(string)
	if !ok {
		return errors.New("remove pin: missing actor")
	}
	target, ok := act["target"].(string)
	if !ok {
		return errors.New("remove pin: missing target")
	}
	switch target {
	case actor + "/collections/featured":
		object, ok := act["object"].(string)
		if !ok {
			return errors.New("remove pin: missing object")
		}
		status, err := models.NewStatuses(i.db).FindByURI(object)
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
		return errors.New("remove pin: unknown target: " + target)
	}
}

func (i *inboxProcessor) processCreate(create map[string]any) error {
	return i.createObject(create)
}

func (i *inboxProcessor) createObject(props map[string]any) error {
	obj := &models.Object{
		Properties: props,
	}
	return i.db.
		Clauses(
			clause.Returning{
				Columns: []clause.Column{{Name: "id"}},
			},
			clause.OnConflict{
				Columns: []clause.Column{{Name: "uri"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"type",
					"properties",
				}),
			}).
		Save(obj).Error
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

// func objToStatusAttachment(obj map[string]any) *models.StatusAttachment {
// 	return &models.StatusAttachment{
// 		Attachment: models.Attachment{
// 			ID:         snowflake.Now(),
// 			MediaType:  stringFromAny(obj["mediaType"]),
// 			URL:        stringFromAny(obj["url"]),
// 			Name:       stringFromAny(obj["name"]),
// 			Width:      intFromAny(obj["width"]),
// 			Height:     intFromAny(obj["height"]),
// 			Blurhash:   stringFromAny(obj["blurhash"]),
// 			FocalPoint: focalPoint(obj),
// 		},
// 	}
// }

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

func (i *inboxProcessor) processFollow(act map[string]any) error {
	uri, ok := act["actor"].(string)
	if !ok {
		return errors.New("follow: actor is not a string")
	}
	actors := models.NewActors(i.db)
	actor, err := actors.FindByURI(uri)
	if err != nil {
		return err
	}
	object, ok := act["object"].(string)
	if !ok {
		return errors.New("follow: object is not a string")
	}
	target, err := actors.FindByURI(object)
	if err != nil {
		return err
	}
	relationships := models.NewRelationships(i.db)
	_, err = relationships.Follow(actor, target)
	return err
}

func (i *inboxProcessor) processUpdate(update map[string]any) error {
	return i.createObject(update)
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

func (i *inboxProcessor) processDelete(act map[string]any) error {
	obj, ok := act["object"]
	if !ok {
		return errors.New("delete: missing object")
	}
	switch obj := obj.(type) {
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
	var obj []models.Object
	if err := i.db.Where("uri = ?", uri).Find(&obj).Error; err != nil {
		return err
	}
	if len(obj) == 0 {
		// already deleted
		return nil
	}
	return i.db.Delete(&obj[0]).Error
}

func (i *inboxProcessor) processDeleteActor(uri string) error {
	// check to see if we have the actor locally, if not, nothing to do.
	var obj []models.Object
	if err := i.db.Where("uri = ?", uri).Find(&obj).Error; err != nil {
		return err
	}
	if len(obj) == 0 {
		// already deleted
		return nil
	}
	if err := i.validateSignature(); err != nil {
		return httpx.Error(http.StatusUnauthorized, err)
	}
	return i.db.Delete(&obj[0]).Error
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
