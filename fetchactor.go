package main

import (
	"context"
	"fmt"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/models"
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

	var props map[string]any
	client, err := activitypub.NewClient(account)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	if err := client.Fetch(context.Background(), f.Actor, &props); err != nil {
		return fmt.Errorf("failed to fetch actor: %w", err)
	}

	return db.Create(&models.Object{Properties: props}).Error
}
