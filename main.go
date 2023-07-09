package main

import (
	"os"

	"golang.org/x/exp/slog"

	"github.com/alecthomas/kong"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Context struct {
	Debug bool

	Logger *slog.Logger

	gorm.Config
	gorm.Dialector
}

var cli struct {
	LogSQL bool   `help:"Log SQL queries."`
	DSN    string `help:"data source name" default:"pub:pub@tcp(localhost:3306)/pub"`

	AutoMigrate          AutoMigrateCmd          `cmd:"" help:"Automigrate the database."`
	CreateAccount        CreateAccountCmd        `cmd:"" help:"Create a new account."`
	CreateInstance       CreateInstanceCmd       `cmd:"" help:"Create a new instance."`
	DeleteAccount        DeleteAccountCmd        `cmd:"" help:"Delete an account."`
	FetchActor           FetchActorCmd           `cmd:"" help:"Fetch an actor."`
	HouseKeeping         HouseKeepingCmd         `cmd:"" help:"Perform housekeeping."`
	Serve                ServeCmd                `cmd:"" help:"Serve a local web server."`
	ShowActor            ShowActorCmd            `cmd:"" help:"Display an actor."`
	SynchroniseFollowers SynchroniseFollowersCmd `cmd:"" help:"Synchronise followers."`
	RerunObjectHooks     RerunObjectHooksCmd     `cmd:"" help:"Rerun object hooks."`
	Follow               FollowCmd               `cmd:"" help:"Follow an object."`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{
		Debug:  cli.LogSQL,
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
		Config: gorm.Config{
			Logger: logger.Default.LogMode(func() logger.LogLevel {
				if cli.LogSQL {
					return logger.Info
				}
				return logger.Warn
			}()),
		},
		Dialector: newDialector(cli.DSN),
	})
	ctx.FatalIfErrorf(err)
}
