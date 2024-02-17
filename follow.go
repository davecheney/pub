package main

import (
	"fmt"
	"os"

	"github.com/davecheney/pub/models"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type FollowCmd struct {
	Object string `help:"object to follow" required:"true"`
	Actor  string `help:"actor to follow with" required:"true"`
}

func (f *FollowCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	actor, err := models.NewActors(db).FindByURI(f.Actor)
	if err != nil {
		return fmt.Errorf("failed to find actor: %w", err)
	}

	object, err := models.NewActors(db).FindOrCreateByURI(f.Object)
	if err != nil {
		return err
	}
	rel, err := models.NewRelationships(db).Follow(actor, object)
	if err != nil {
		return err
	}
	json.MarshalWrite(os.Stdout, rel)
	return nil
}
