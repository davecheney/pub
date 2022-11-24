package main

import (
	"context"

	"github.com/davecheney/m/mastodon"
	"github.com/jmoiron/sqlx"
)

type InboxCmd struct {
	DSN string `help:"data source name"`
}

func (i *InboxCmd) Run(ctx *Context) error {
	db, err := sqlx.Connect("mysql", i.DSN+"?parseTime=true")
	if err != nil {
		return err
	}
	ip := mastodon.NewInboxProcessor(db)
	return ip.Run(context.Background())
}
