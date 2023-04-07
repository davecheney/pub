package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/activitypub"
	"github.com/davecheney/pub/internal/models"
	"gorm.io/gorm"
)

// RelationshipRequestProcessor handles delivery of relationship requests.
func NewRelationshipRequestProcessor(db *gorm.DB) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		fmt.Println("RelationshipRequestProcessor started")
		defer fmt.Println("RelationshipRequestProcessor stopped")

		db := db.WithContext(ctx)
		for {
			if err := process(db, relationshipRequestScope, processRelationshipRequest); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(30 * time.Second):
				// continue
			}
		}
	}
}

func relationshipRequestScope(db *gorm.DB) *gorm.DB {
	return db.Preload("Actor").Preload("Target").Where("attempts < 3")
}

func processRelationshipRequest(db *gorm.DB, request *models.RelationshipRequest) error {
	fmt.Println("RelationshipRequestProcessor: actor:", request.Actor.URI, "target:", request.Target.URI, "action:", request.Action)

	accounts := models.NewAccounts(db)
	account, err := accounts.AccountForActor(request.Actor)
	if err != nil {
		return err
	}

	client, err := activitypub.NewClient(db.Statement.Context, account)
	if err != nil {
		return err
	}

	switch request.Action {
	case "follow":
		return client.Follow(account.Actor.URI, request.Target.URI)
	case "unfollow":
		return client.Unfollow(account.Actor.URI, request.Target.URI)
	default:
		return fmt.Errorf("unknown action %q", request.Action)
	}
}
