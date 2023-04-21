package main

import (
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

type CreateAccountCmd struct {
	Name     string `required:"" help:"name of the account to create"`
	Domain   string `required:"" help:"domain of the account to create"`
	Email    string `required:"" help:"email address of the account to create"`
	Password string `required:"" help:"password of the account to create"`
}

func (c *CreateAccountCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	var instance models.Instance
	if err := db.Where("domain = ?", c.Domain).First(&instance).Error; err != nil {
		return err
	}

	_, err = models.NewAccounts(db).Create(&instance, c.Name, c.Email, c.Password)
	return err
}
