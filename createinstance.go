package main

import (
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

type CreateInstanceCmd struct {
	Domain      string `required:"" help:"domain name of the instance to create"`
	Title       string `required:"" help:"title of the instance to create"`
	Description string `required:"" help:"description of the instance to create"`
	AdminEmail  string `required:"" help:"email address of the admin account to create"`
}

func (c *CreateInstanceCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}
	_, err = models.NewInstances(db).Create(c.Domain, c.Title, c.Description, c.AdminEmail)
	return err

}

func withTransaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	tx := db.Begin()
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
