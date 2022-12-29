package activitypub

import (
	"crypto"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"github.com/davecheney/m/m"
	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Inboxes struct {
	service *Service
	getKey  func(keyId string) (crypto.PublicKey, error)
}

func (i *Inboxes) Create(w http.ResponseWriter, r *http.Request) {
	if err := i.validateSignature(r); err != nil {
		fmt.Println("validateSignature failed", err)
	}

	var body map[string]any
	if err := json.UnmarshalFull(r.Body, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := i.processActivity(body)
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
func (i *Inboxes) processActivity(body map[string]any) error {
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
	svc := m.NewService(i.service.db)
	status, err := svc.Statuses().FindByURI(id)
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
	svc := m.NewService(i.service.db)
	actor, err := svc.Actors().FindByURI(stringFromAny(body["actor"]))
	if err != nil {
		return err
	}
	target, err := svc.Actors().FindByURI(stringFromAny(body["object"]))
	if err != nil {
		return err
	}
	_, err = svc.Relationships().Unfollow(actor, target)
	return err
}

func (i *Inboxes) processAnnounce(obj map[string]any) error {
	target := stringFromAny(obj["object"])
	svc := m.NewService(i.service.db)
	original, err := svc.Statuses().FindOrCreate(target, svc.Statuses().NewRemoteStatusFetcher().Fetch)
	if err != nil {
		return err
	}

	actor, err := svc.Actors().FindOrCreate(stringFromAny(obj["actor"]), svc.Actors().NewRemoteActorFetcher().Fetch)
	if err != nil {
		return err
	}

	published, err := timeFromAny(obj["published"])
	if err != nil {
		return err
	}

	conv := m.Conversation{
		Visibility: "public",
	}
	if err := i.service.db.Create(&conv).Error; err != nil {
		return err
	}

	status := &m.Status{
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
		Reblog:           original,
	}

	return i.service.db.Create(status).Error
}

func (i *Inboxes) processAdd(act map[string]any) error {
	target := stringFromAny(act["target"])
	switch target {
	case stringFromAny(act["actor"]) + "/collections/featured":
		svc := m.NewService(i.service.db)
		status, err := svc.Statuses().FindByURI(stringFromAny(act["object"]))
		if err != nil {
			return err
		}
		actor, err := svc.Actors().FindByURI(stringFromAny(act["actor"]))
		if err != nil {
			return err
		}
		if actor.ID != status.ActorID {
			return errors.New("actor is not the author of the status")
		}
		return svc.Reactions().Pin(status, actor)
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
		svc := m.NewService(i.service.db)
		status, err := svc.Statuses().FindByURI(stringFromAny(act["object"]))
		if err != nil {
			return err
		}
		actor, err := svc.Actors().FindByURI(stringFromAny(act["actor"]))
		if err != nil {
			return err
		}
		if actor.ID != status.ActorID {
			return errors.New("actor is not the author of the status")
		}
		return svc.Reactions().Unpin(status, actor)
	default:
		x, _ := marshalIndent(act)
		fmt.Println("processRemove:", string(x))
		return errors.New("not implemented")
	}
}

func (i *Inboxes) processCreate(create map[string]any) error {
	typ := stringFromAny(create["type"])
	switch typ {
	case "Note":
		return i.processCreateNote(create)
	default:
		return fmt.Errorf("unknown create object type: %q", typ)
	}
}

func (i *Inboxes) processCreateNote(create map[string]any) error {
	uri := stringFromAny(create["atomUri"])
	if uri == "" {
		return errors.New("missing atomUri")
	}

	svc := m.NewService(i.service.db)
	_, err := svc.Statuses().FindOrCreate(uri, func(string) (*m.Status, error) {
		fetcher := svc.Actors().NewRemoteActorFetcher()
		actor, err := svc.Actors().FindOrCreate(stringFromAny(create["attributedTo"]), fetcher.Fetch)
		if err != nil {
			return nil, err
		}

		published, err := timeFromAny(create["published"])
		if err != nil {
			return nil, err
		}

		var inReplyTo *m.Status
		if inReplyToAtomUri, ok := create["inReplyTo"].(string); ok {
			remoteStatusFetcher := svc.Statuses().NewRemoteStatusFetcher()
			inReplyTo, err = svc.Statuses().FindOrCreate(inReplyToAtomUri, remoteStatusFetcher.Fetch)
			if err != nil {
				fmt.Println("inReplyToAtomUri:", inReplyToAtomUri, "err:", err)
			}
		}

		vis := visiblity(create)
		conversationID := uint32(0)
		if inReplyTo != nil {
			conversationID = inReplyTo.ConversationID
		} else {
			conv, err := svc.Conversations().New(vis)
			if err != nil {
				return nil, err
			}
			conversationID = conv.ID
		}

		st := &m.Status{
			ID:               snowflake.TimeToID(published),
			ActorID:          actor.ID,
			Actor:            actor,
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
			fmt.Println("attchment:", at)
			st.Attachments = append(st.Attachments, m.StatusAttachment{
				Attachment: m.Attachment{
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

func inReplyToID(inReplyTo *m.Status) *uint64 {
	if inReplyTo != nil {
		return &inReplyTo.ID
	}
	return nil
}

func inReplyToActorID(inReplyTo *m.Status) *uint64 {
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
	svc := m.NewService(i.service.db)
	actor, err := svc.Actors().FindByURI(stringFromAny(body["actor"]))
	if err != nil {
		return err
	}
	target, err := svc.Actors().FindByURI(stringFromAny(body["object"]))
	if err != nil {
		return err
	}
	_, err = svc.Relationships().Follow(actor, target)
	return err
}

func (i *Inboxes) processUpdate(update map[string]any) error {
	id := stringFromAny(update["id"])
	var status m.Status
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
	actor := stringFromAny(body["object"])
	err := i.service.db.Where("uri = ?", actor).Delete(&m.Actor{}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// already deleted
		return nil
	}
	return err
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

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func timeFromAny(v any) (time.Time, error) {
	switch v := v.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	case time.Time:
		return v, nil
	default:
		return time.Time{}, errors.New("timeFromAny: invalid type")
	}
}

func intFromAny(v any) int {
	switch v := v.(type) {
	case int:
		return v
	case float64:
		// shakes fist at json number type
		return int(v)
	}
	return 0
}

func anyToSlice(v any) []any {
	switch v := v.(type) {
	case []any:
		return v
	default:
		return nil
	}
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
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

func marshalIndent(v any) ([]byte, error) {
	b, err := json.MarshalOptions{}.Marshal(json.EncodeOptions{
		Indent: "\t", // indent for readability
	}, v)
	return b, err
}
