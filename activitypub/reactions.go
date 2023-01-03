package activitypub

import (
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/activitypub"
	"github.com/davecheney/pub/internal/models"
	"gorm.io/gorm"
)

// ReactionRequestProcessor handles delivery of relationship requests.
type ReactionRequestProcessor struct {
	db *gorm.DB
}

func NewReactionRequestProcessor(db *gorm.DB) *ReactionRequestProcessor {
	return &ReactionRequestProcessor{
		db: db,
	}
}

func (rrp *ReactionRequestProcessor) Run(stop <-chan struct{}) error {
	fmt.Println("ReactionRequestProcessor.Run started")
	defer fmt.Println("ReactionRequestProcessor.Run stopped")

	for {
		if err := rrp.process(); err != nil {
			return err
		}
		select {
		case <-stop:
			return nil
		case <-time.After(30 * time.Second):
			// continue
		}
	}
}

// process make one pass through the RelationshipRequest table, processing
// any pending requests.
func (rrp *ReactionRequestProcessor) process() error {
	var requests []models.ReactionRequest
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

func (rrp *ReactionRequestProcessor) processRequest(request *models.ReactionRequest) error {
	fmt.Println("ReactionRequestProcessor.processRequest: actor:", request.Actor.URI, "target:", request.Target.URI, "action:", request.Action)

	accounts := models.NewAccounts(rrp.db)
	account, err := accounts.AccountForActor(request.Actor)
	if err != nil {
		return err
	}

	switch request.Action {
	case "like":
		return rrp.processLikeRequest(account, request.Target)
	case "unlike":
		return rrp.processUnlikeRequest(account, request.Target)
	default:
		return fmt.Errorf("unknown action %q", request.Action)
	}
}

func (rrp *ReactionRequestProcessor) processLikeRequest(account *models.Account, target *models.Status) error {
	client, err := activitypub.NewClient(rrp.db.Statement.Context, account)
	if err != nil {
		return err
	}
	return client.Like(account.Actor.URI, target.URI)
}

func (rrp *ReactionRequestProcessor) processUnlikeRequest(account *models.Account, target *models.Status) error {
	client, err := activitypub.NewClient(rrp.db.Statement.Context, account)
	if err != nil {
		return err
	}
	return client.Unlike(account.Actor.URI, target.URI)
}
