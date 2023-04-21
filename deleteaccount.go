package main

import (
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

type DeleteAccountCmd struct {
	Name   string `required:"" help:"name of the account to delete"`
	Domain string `required:"" help:"domain of the account to delete"`
}

func (d *DeleteAccountCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	return db.Where("name = ? AND domain = ?", d.Name, d.Domain).Delete(&models.Account{}).Error
}
