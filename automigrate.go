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

	ctx.Logger.Info("setting all in_reply_to_id to null if the parent status does not exist")
	var statuses []*models.Status
	err = db.Preload("Actor").Where("in_reply_to_id is not null").FindInBatches(&statuses, 1000, func(tx *gorm.DB, batch int) error {
		for _, status := range statuses {
			var parent models.Status
			if err := tx.Preload("Actor").Where("statuses.id = ?", status.InReplyToID).First(&parent).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					status.InReplyToID = nil
					status.InReplyToActorID = nil
					if err := tx.Save(status).Error; err != nil {
						return err
					}
				} else {
					return err
				}
			}
		}
		return nil
	}).Error
	if err != nil {
		return err
	}

	ctx.Logger.Info("apply migrations")
	if err := db.AutoMigrate(models.AllTables()...); err != nil {
		return err
	}
	ctx.Logger.Info("migration complete")

	// post migration fixups

	ctx.Logger.Info("converting admin account to a LocalService")
	err = db.Model(&models.Actor{}).Where("type = ? and name = ?", "Service", "admin").UpdateColumn("type", "LocalService").Error
	if err != nil {
		return err
	}

	return nil
}
