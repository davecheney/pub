package main

import (
	"github.com/davecheney/m/m"
	"gorm.io/gorm"
)

type CreateInstanceCmd struct {
	Domain      string `required:"" help:"domain name of the instance to create"`
	Title       string `required:"" help:"title of the instance to create"`
	Description string `required:"" help:"description of the instance to create"`
	Admin       bool   `help:"create an admin account"`
}

func (c *CreateInstanceCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	instance := &m.Instance{
		Domain:      c.Domain,
		Title:       c.Title,
		Description: c.Description,
	}
	return db.Create(instance).Error
}
