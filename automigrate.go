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

	if err := db.AutoMigrate(&mastodon.Instance{}, &mastodon.InstanceRule{}); err != nil {
		return err
	}

	return db.AutoMigrate(
		&activitypub.Activity{},

		&mastodon.Account{},
		&mastodon.Application{},
		&mastodon.ClientFilter{},

		&mastodon.Notification{},
		&mastodon.Status{},
		&mastodon.Token{},
	)
}
