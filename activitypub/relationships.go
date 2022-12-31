package activitypub

import (
	"fmt"
	"time"

	"github.com/davecheney/m/internal/activitypub"
	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

// RelationshipRequestProcessor handles delivery of relationship requests.
type RelationshipRequestProcessor struct {
	db *gorm.DB
}

func NewRelationshipRequestProcessor(db *gorm.DB) *RelationshipRequestProcessor {
	return &RelationshipRequestProcessor{
		db: db,
	}
}

func (rrp *RelationshipRequestProcessor) Run(stop <-chan struct{}) error {
	fmt.Println("RelationshipRequestProcessor.Run started")
	defer fmt.Println("RelationshipRequestProcessor.Run stopped")

	if err := rrp.process(); err != nil {
		return err
	}

	for {
		select {
		case <-stop:
			return nil
		case <-time.After(30 * time.Second):
			if err := rrp.process(); err != nil {
				return err
			}
		}
	}
}

// process make one pass through the RelationshipRequest table, processing
// any pending requests.
func (rrp *RelationshipRequestProcessor) process() error {
	var requests []models.RelationshipRequest
	if err := rrp.db.Preload("Actor").Preload("Target").Find(&requests).Error; err != nil {
		return err
	}

	for _, request := range requests {
		if err := rrp.processRequest(&request); err != nil {
			request.LastAttempt = time.Now()
			request.Attempts++
			request.LastResult = err.Error()
			if err := rrp.db.Save(&request).Error; err != nil {
				return err
			}
		}
		if err := rrp.db.Delete(&request).Error; err != nil {
			return err
		}
	}

	return nil
}

func (rrp *RelationshipRequestProcessor) processRequest(request *models.RelationshipRequest) error {
	fmt.Println("RelationshipRequestProcessor.processRequest: actor:", request.Actor.URI, "target:", request.Target.URI, "action:", request.Action)

	accounts := models.NewAccounts(rrp.db)
	account, err := accounts.AccountForActor(request.Actor)
	if err != nil {
		return err
	}

	switch request.Action {
	case "follow":
		return rrp.processFollowRequest(account, request.Target)
	case "unfollow":
		return rrp.processUnfollowRequest(account, request.Target)
	default:
		return fmt.Errorf("unknown action %q", request.Action)
	}
}

func (rrp *RelationshipRequestProcessor) processFollowRequest(account *models.Account, target *models.Actor) error {
	client, err := activitypub.NewClient(account.Actor.PublicKeyID(), account.PrivateKey)
	if err != nil {
		return err
	}
	return client.Follow(account.Actor.URI, target.URI)
}

func (rrp *RelationshipRequestProcessor) processUnfollowRequest(account *models.Account, target *models.Actor) error {
	client, err := activitypub.NewClient(account.Actor.PublicKeyID(), account.PrivateKey)
	if err != nil {
		return err
	}
	return client.Unfollow(account.Actor.URI, target.URI)
}
