package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/davecheney/pub/mastodon"
	"github.com/davecheney/pub/models"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type ShowActorCmd struct {
	Actor string `required:"" help:"The actor uri to display."`
}

func (s *ShowActorCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	actor, err := models.NewActors(db).FindByURI(s.Actor)
	if err != nil {
		return fmt.Errorf("failed to find actor %s: %w", s.Actor, err)
	}

	req, _ := http.NewRequest("GET", actor.URI(), nil)
	ser := mastodon.NewSerialiser(req)
	return json.MarshalWrite(os.Stdout, ser.Account(actor))
}
