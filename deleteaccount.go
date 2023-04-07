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

	return withTransaction(db, func(tx *gorm.DB) error {
		var account models.Account
		if err := tx.Joins("Actor").First(&account, "name = ? AND domain = ?", d.Name, d.Domain).Error; err != nil {
			return err
		}

		// delete the actor
		if err := tx.Delete(&account.Actor).Error; err != nil {
			return err
		}

		// delete the account
		return tx.Delete(&account).Error
	})

}
