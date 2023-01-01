package main

import (
	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

type AutoMigrateCmd struct {
}

func (a *AutoMigrateCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	return db.AutoMigrate(
		&models.Actor{},
		&models.Account{}, &models.AccountList{}, &models.AccountListMember{}, &models.AccountRole{}, &models.AccountMarker{},
		&models.Application{},
		&models.Conversation{},
		&models.Instance{}, &models.InstanceRule{},
		&models.Reaction{}, &models.ReactionRequest{},
		&models.Relationship{}, &models.RelationshipRequest{},
		// &models.Notification{},
		&models.Status{}, &models.StatusPoll{}, &models.StatusAttachment{},
		&models.Token{},
	)
}
