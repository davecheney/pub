package main

import (
	"context"

	"github.com/davecheney/pub/internal/activitypub"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

type FollowCmd struct {
	Object string `help:"object to follow" required:"true"`
	Actor  string `help:"actor to follow with" required:"true"`
}

func (f *FollowCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	var account models.Account
	if err := db.Joins("Actor", &models.Actor{URI: f.Actor}).Take(&account).Error; err != nil {
		return err
	}

	client, err := activitypub.NewClient(context.Background(), &account)
	if err != nil {
		return err
	}

	return client.Follow(account.Actor.URI, f.Object)
}
