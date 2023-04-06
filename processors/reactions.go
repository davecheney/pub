package processors

import (
	"context"
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/activitypub"
	"github.com/davecheney/pub/internal/models"
	"gorm.io/gorm"
)

// NewReactionRequestProcessor handles delivery of relationship requests.
func NewReactionRequestProcessor(db *gorm.DB) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		fmt.Println("ReactionRequestProcessor started")
		defer fmt.Println("ReactionRequestProcessor stopped")

		db := db.WithContext(ctx)
		for {
			if err := process(db, reactionRequestScope, processReactionRequest); err != nil {
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

func reactionRequestScope(db *gorm.DB) *gorm.DB {
	return db.Preload("Actor").Preload("Target").Where("attempts < 3")
}

func processReactionRequest(db *gorm.DB, request *models.ReactionRequest) error {
	fmt.Println("ReactionRequestProcessor: actor:", request.Actor.URI, "target:", request.Target.URI, "action:", request.Action)

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
	case "like":
		return client.Like(account.Actor.URI, request.Target.URI)
	case "unlike":
		return client.Unlike(account.Actor.URI, request.Target.URI)
	default:
		return fmt.Errorf("unknown action %q", request.Action)
	}
}
