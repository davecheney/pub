package main

import (
	"os"

	"github.com/davecheney/m/mastodon"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type CreateInstanceCmd struct {
	Domain string `help:"domain name of the instance to create"`
}

func (c *CreateInstanceCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}
	instance := &mastodon.Instance{
		Domain: c.Domain,
	}
	if err := db.Create(instance).Error; err != nil {
		return err
	}
	return json.MarshalFull(os.Stdout, instance)
}
