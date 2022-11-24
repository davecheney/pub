package main

import (
	"github.com/alecthomas/kong"

	_ "github.com/go-sql-driver/mysql"
)

type Context struct {
	Debug bool
}

var cli struct {
	Debug bool `help:"Enable debug mode."`

	Serve  ServeCmd  `cmd:"" help:"Serve a local web server."`
	Inbox  InboxCmd  `cmd:"" help:"Process activitypub inbox."`
	Follow FollowCmd `cmd:"" help:"Follow an object."`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
