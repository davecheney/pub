package main

import (
	"github.com/alecthomas/kong"
	"gorm.io/gorm"

	_ "github.com/go-sql-driver/mysql"
)

type Context struct {
	Debug bool

	gorm.Config
}

var cli struct {
	Debug bool `help:"Enable debug mode."`

	Serve  ServeCmd  `cmd:"" help:"Serve a local web server."`
	Inbox  IndexCmd  `cmd:"" help:"Process the inbox."`
	Follow FollowCmd `cmd:"" help:"Follow an object."`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
