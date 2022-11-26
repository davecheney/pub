package main

import (
	"github.com/davecheney/m/mastodon"
	"gorm.io/gorm"
)

type CreateInstanceCmd struct {
	Domain string `required:"" help:"domain name of the instance to create"`
}

func (c *CreateInstanceCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}
	instance := &mastodon.Instance{
		Domain: c.Domain,
	}
	return db.Create(instance).Error
}
