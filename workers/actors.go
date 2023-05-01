package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/webfinger"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

// NewActorRefreshProcessor handles updating the actor's record.
func NewActorRefreshProcessor(db *gorm.DB, admin *models.Account) func(ctx context.Context) error {

	return func(ctx context.Context) error {
		fmt.Println("NewActorRefreshProcessor started")
		defer fmt.Println("NewActorRefreshProcessor stopped")

		refresher := &actorRefresher{
			signAs: admin,
		}

		db := db.WithContext(ctx)
		for {
			if err := process(db, actorRefreshScope, refresher.processActorRefresh); err != nil {
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

func actorRefreshScope(db *gorm.DB) *gorm.DB {
	return db.Preload("Actor").Preload("Actor.Attributes").Where("attempts < 3")
}

type actorRefresher struct {
	// signAs is the account to sign requests as.
	signAs *models.Account
}

func (a *actorRefresher) processActorRefresh(db *gorm.DB, request *models.ActorRefreshRequest) error {
	if request.Actor.IsLocal() {
		// ignore local actors
		return nil
	}
	ctx := db.Statement.Context
	acct := webfinger.Acct{
		User: request.Actor.Name,
		Host: request.Actor.Domain,
	}
	fmt.Println("processActorRefresh", acct.String())
	var finger webfinger.Webfinger
	if err := requests.URL(acct.Webfinger()).ToJSON(&finger).Fetch(ctx); err != nil {
		return err
	}
	ap, err := finger.ActivityPub()
	if err != nil {
		return err
	}

	orig := request.Actor
	updated, err := activitypub.NewRemoteActorFetcher(a.signAs, db).Fetch(ap)
	if err != nil {
		return err
	}

	// RemoteActorFetcher.Fetch will have created a new snowflake ID for the updated record
	// even if the created-at date has not changed because of the random component of the ID.
	// We need to update the ID to match the original record.
	updated.ID = orig.ID
	return db.Transaction(func(tx *gorm.DB) error {
		// delete actor attributes
		if err := tx.Where("actor_id = ?", orig.ID).Delete(&models.ActorAttribute{}).Error; err != nil {
			return err
		}
		// save updated actor
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Updates(updated).Error
	})
}
