package main

import (
	"context"
	"io"
	"os"
	"time"

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
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	err := ctx.Run(&Context{
		Debug:  cli.LogSQL,
		Logger: log,
		Config: gorm.Config{
			Logger: &slogGORMLogger{slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: func() slog.Level {
					if cli.LogSQL {
						return slog.LevelInfo
					}
					return slog.LevelWarn
				}(),
			}))},
		},
		Dialector: newDialector(cli.DSN),
	})
	ctx.FatalIfErrorf(err)
}

type slogHandler struct {
	out io.Writer
}

type slogGORMLogger struct {
	*slog.Logger
}

func (s *slogGORMLogger) LogMode(level logger.LogLevel) logger.Interface {
	return s
}

func (s *slogGORMLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	s.InfoContext(ctx, msg, data...)
}

func (s *slogGORMLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	s.WarnContext(ctx, msg, data...)
}

func (s *slogGORMLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	s.ErrorContext(ctx, msg, data...)
}

func (s *slogGORMLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rowsAffected := fc()
	s.InfoContext(ctx, "pub/sql.trace", "sql", sql, "rowsAffected", rowsAffected, "err", err)
}
