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
	DSN   string `help:"data source name" default:"m:m@tcp(localhost:3306)/m"`

	AutoMigrate          AutoMigrateCmd          `cmd:"" help:"Automigrate the database."`
	CreateAccount        CreateAccountCmd        `cmd:"" help:"Create a new account."`
	CreateInstance       CreateInstanceCmd       `cmd:"" help:"Create a new instance."`
	DeleteAccount        DeleteAccountCmd        `cmd:"" help:"Delete an account."`
	Serve                ServeCmd                `cmd:"" help:"Serve a local web server."`
	SynchroniseFollowers SynchroniseFollowersCmd `cmd:"" help:"Synchronise followers."`
	Imoport              ImportCmd               `cmd:"" help:"Import data from another instance."`
	Follow               FollowCmd               `cmd:"" help:"Follow an object."`
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
		Dialector: mysql.New(mysql.Config{
			DSN:                       cli.DSN + "?charset=utf8mb4&parseTime=True&loc=Local",
			SkipInitializeWithVersion: false, // auto configure based on currently MySQL version
		}),
	})
	ctx.FatalIfErrorf(err)
}
