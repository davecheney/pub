package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/models"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

// RelationshipRequestProcessor handles delivery of relationship requests.
func NewRelationshipRequestProcessor(log *slog.Logger, db *gorm.DB) func(ctx context.Context) error {
	log = log.With("worker", "RelationshipRequestProcessor")
	return func(ctx context.Context) error {
		log.Info("started")
		defer log.Info("stopped")

		db := db.WithContext(ctx)
		for {
			if err := process(db, relationshipRequestScope, func(db *gorm.DB, request *models.RelationshipRequest) error {
				return processRelationshipRequest(log, db, request)
			}); err != nil {
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

func processRelationshipRequest(log *slog.Logger, db *gorm.DB, request *models.RelationshipRequest) error {
	log.Info("processRelationshipRequest", "request", request.ID, "actor_id", request.Actor.ObjectID, "target_id", request.Target.ObjectID, "action", request.Action)
	accounts := models.NewAccounts(db)
	account, err := accounts.AccountForActor(request.Actor)
	if err != nil {
		return err
	}
	switch request.Action {
	case "follow":
		return activitypub.Follow(db.Statement.Context, account, request.Target)
	case "unfollow":
		return activitypub.Unfollow(db.Statement.Context, account, request.Target)
	default:
		return fmt.Errorf("unknown action %q", request.Action)
	}
}
