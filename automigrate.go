package main

import (
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

type AutoMigrateCmd struct {
	DisableForeignKeyConstraints bool `help:"disable foreign key constraints when creating tables."`
}

func (a *AutoMigrateCmd) Run(ctx *Context) error {
	ctx.Config.DisableForeignKeyConstraintWhenMigrating = a.DisableForeignKeyConstraints
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	if err := db.AutoMigrate(models.AllTables()...); err != nil {
		return err
	}

	// post migration fixups

	// convert the admin account to a LocalService
	err = db.Model(&models.Actor{}).Where("type = ? and name = ?", "Service", "admin").UpdateColumn("type", "LocalService").Error
	if err != nil {
		return err
	}

	return nil
}
