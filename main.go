package main

import (
	"github.com/alecthomas/kong"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/go-sql-driver/mysql"
)

type Context struct {
	Debug bool

	gorm.Config
	gorm.Dialector
}

var cli struct {
	Debug bool   `help:"Enable debug mode."`
	DSN   string `help:"data source name"`

	AutoMigrate AutoMigrateCmd `cmd:"" help:"Automigrate the database."`
	Serve       ServeCmd       `cmd:"" help:"Serve a local web server."`
	Inbox       IndexCmd       `cmd:"" help:"Process the inbox."`
	Follow      FollowCmd      `cmd:"" help:"Follow an object."`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{
		Debug: cli.Debug,
		Config: gorm.Config{
			Logger: logger.Default.LogMode(func() logger.LogLevel {
				if cli.Debug {
					return logger.Info
				}
				return logger.Warn
			}()),
		},
		Dialector: mysql.Open(cli.DSN + "?charset=utf8mb4&parseTime=True&loc=Local"),
	})
	ctx.FatalIfErrorf(err)
}
