package activitypub

import (
	"crypto"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/m/m"
	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Activity struct {
	gorm.Model
	Object map[string]interface{} `gorm:"serializer:json"`
}

func (Activity) TableName() string {
	return "inbox"
}

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
	typ := stringFromAny(body["type"])
	switch typ {
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
	default:
		id := stringFromAny(body["id"])
		fmt.Println("processActivity: queuing activity", id)
		// Too hard, queue it.
		activity := Activity{
			Object: body,
		}
		return i.service.db.Create(&activity).Error
	}
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
	var actor m.Actor
	if err := i.service.db.First(&actor, "actor_id = ?", stringFromAny(body["actor"])).Error; err != nil {
		return err
	}
	var target m.Actor
	if err := i.service.db.First(&target, "actor_id = ?", stringFromAny(body["object"])).Error; err != nil {
		return err
	}

	var rel m.Relationship
	if err := i.service.db.Joins("Target").First(&rel, "actor_id = ? and target_id = ?", actor.ID, target.ID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		rel = m.Relationship{
			ActorID:  actor.ID,
			TargetID: target.ID,
			Target:   &target,
		}
	}

	rel.FollowedBy = true
	return i.service.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&rel).Error
}

func (i *Inboxes) processUpdate(update map[string]any) error {
	id := stringFromAny(update["id"])
	var status m.Status
	if err := i.service.db.Where("uri = ?", id).First(&status).Error; err != nil {
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
	fmt.Println("processDelete", body)
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
