package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/mastodon"
	"github.com/davecheney/pub/models"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type FetchActorCmd struct {
	Account string `required:"" help:"The account to sign the request with."`
	Actor   string `required:"" help:"The actor to fetch."`
}

func (f *FetchActorCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	signAs, err := models.NewActors(db).FindByURI(f.Account)
	if err != nil {
		return fmt.Errorf("failed to find actor: %w", err)
	}
	account, err := models.NewAccounts(db).AccountForActor(signAs)
	if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}

	orig, err := models.NewActors(db).FindByURI(f.Actor)
	if err != nil {
		return fmt.Errorf("failed to find actor: %w", err)
	}

	updated, err := activitypub.NewRemoteActorFetcher(account, db).Fetch(f.Actor)
	if err != nil {
		return fmt.Errorf("failed to fetch actor: %w", err)
	}

	if !orig.ID.ToTime().Equal(updated.ID.ToTime()) {
		return fmt.Errorf("actor ID changed from %v to %v", orig.ID, updated.ID)
	}

	// RemoteActorFetcher.Fetch will have created a new snowflake ID for the updated record
	// even if the created-at date has not changed because of the random component of the ID.
	// We need to update the ID to match the original record.
	updated.ID = orig.ID

	req, _ := http.NewRequest("GET", updated.URI, nil)
	ser := mastodon.NewSerialiser(req)
	json.MarshalFull(os.Stdout, map[string]any{
		"original": ser.Account(orig),
		"updated":  ser.Account(updated),
	})

	return db.Transaction(func(tx *gorm.DB) error {
		// delete actor attributes
		if err := tx.Where("actor_id = ?", orig.ID).Delete(&models.ActorAttribute{}).Error; err != nil {
			return err
		}
		// save updated actor
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Updates(updated).Error
	})
}
