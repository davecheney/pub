package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/models"
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
	return db.Preload("Actor").Preload("Target").Preload("Target.Actor").Where("attempts < 3")
}

func processReactionRequest(db *gorm.DB, request *models.ReactionRequest) error {
	fmt.Println("ReactionRequestProcessor: actor:", request.Actor.URI, "target:", request.Target.URI, "action:", request.Action)

	accounts := models.NewAccounts(db)
	account, err := accounts.AccountForActor(request.Actor)
	if err != nil {
		return err
	}

	inbox := request.Target.Actor.Inbox()
	if inbox == "" {
		if err := models.NewActors(db).Refresh(request.Target.Actor); err != nil {
			return err
		}
		return fmt.Errorf("no inbox for actor %q", request.Target.Actor.URI)
	}

	switch request.Action {
	case "like":
		return activitypub.Like(db.Statement.Context, account, request.Target)
	case "unlike":
		return activitypub.Unlike(db.Statement.Context, account, request.Target)
	default:
		return fmt.Errorf("unknown action %q", request.Action)
	}
}
