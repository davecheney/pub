package main

import (
	"github.com/davecheney/m/m"
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
		&m.Actor{}, &m.Webfinger{},
		&m.Account{}, &m.AccountList{},
		&m.Application{},
		&m.Conversation{},
		&m.ClientFilter{},
		&m.Instance{}, &m.InstanceRule{},
		&m.Relationship{},
		&m.Marker{},
		&m.Notification{},
		&m.Status{}, &m.Poll{}, &m.StatusAttachment{},
		&m.Token{},
	)
}
