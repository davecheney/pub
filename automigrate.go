package main

import (
	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/mastodon"
	"gorm.io/gorm"
)

type AutoMigrateCmd struct {
}

func (a *AutoMigrateCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	return db.AutoMigrate(
		&mastodon.Status{},
		&mastodon.User{},
		&mastodon.Account{},
		&mastodon.Application{},
		&mastodon.Token{},

		&activitypub.Actor{},
		&activitypub.Activity{},
	)
}
