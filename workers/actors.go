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

		c, err := activitypub.NewClient(ctx, admin)
		if err != nil {
			return err
		}

		refresher := &actorRefresher{
			client: c,
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
	// client is the client used to fetch the actor's inbox and outbox URLs.
	client *activitypub.Client
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

	var actor activitypub.Actor
	if err := a.client.Fetch(ap, &actor); err != nil {
		return err
	}
	return db.Model(request.Actor).UpdateColumns(map[string]interface{}{
		"inbox_url":        actor.Inbox,
		"outbox_url":       actor.Outbox,
		"shared_inbox_url": actor.Endpoints.SharedInbox,
	}).Error
}
