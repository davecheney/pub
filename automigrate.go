package main

import (
	"github.com/davecheney/pub/internal/models"
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

	err = db.AutoMigrate(
		&models.ActivitypubRefresh{},
		&models.Actor{}, &models.ActorAttribute{},
		&models.Account{}, &models.AccountList{}, &models.AccountListMember{}, &models.AccountRole{}, &models.AccountMarker{}, &models.AccountPreferences{},
		&models.Application{},
		&models.Conversation{},
		&models.Instance{}, &models.InstanceRule{},
		&models.Reaction{}, &models.ReactionRequest{},
		&models.Relationship{}, &models.RelationshipRequest{},
		// &models.Notification{},
		&models.Status{}, &models.StatusPoll{}, &models.StatusPollOption{}, &models.StatusAttachment{}, &models.StatusMention{}, &models.StatusTag{},
		&models.StatusAttachmentRequest{},
		&models.Tag{},
		&models.Token{},
	)
	if err != nil {
		return err
	}

	// post migration fixups

	// convert the admin account to a LocalService
	err = db.Model(&models.Actor{}).Where("type = ? and name = ?", "Service", "admin").Update("type", "LocalService").Error
	if err != nil {
		return err
	}

	// load each status attachment and save it to trigger the AfterSave hook
	var results []models.StatusAttachment
	return db.FindInBatches(&results, 100, func(tx *gorm.DB, batch int) error {
		return tx.Save(&results).Error
	}).Error
}
