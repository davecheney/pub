package main

import (
	"net/http"
	"time"

	"github.com/alecthomas/kong"
	"github.com/davecheney/m/m"
	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

type Context struct {
	Debug bool
}

type ServeCmd struct {
	Addr string `help:"address to listen"`
	DSN  string `help:"data source name"`
}

func (s *ServeCmd) Run(ctx *Context) error {
	db, err := sqlx.Connect("mysql", s.DSN)
	if err != nil {
		return err
	}
	svr := &http.Server{
		Addr:         s.Addr,
		Handler:      m.New(db),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return svr.ListenAndServe()
}

var cli struct {
	Debug bool `help:"Enable debug mode."`

	Serve ServeCmd `cmd:"" help:"Remove files."`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
