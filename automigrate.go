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
		&m.Account{}, &m.AccountList{}, &m.LocalAccount{},
		&m.Activity{},
		&m.Application{},
		&m.Conversation{},
		&m.ClientFilter{},
		&m.Instance{}, &m.InstanceRule{},
		&m.Marker{},
		&m.Notification{},
		&m.Status{},
		&m.Token{},
	)
}
