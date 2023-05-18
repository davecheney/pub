package main

import (
	"context"
	"sync"

	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

type RerunObjectHooksCmd struct {
	Domain string `required:"" help:"domain name of the instance to backfill from."`
}

func (c *RerunObjectHooksCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	instance, err := models.NewInstances(db).FindByDomain(c.Domain)
	if err != nil {
		return err
	}

	var objs []*models.Object
	db = db.WithContext(
		context.WithValue(db.Statement.Context, "instance", instance),
	)
	return db.FindInBatches(
		&objs, 25, func(tx *gorm.DB, batch int) error {
			var wg sync.WaitGroup
			wg.Add(len(objs))
			for _, obj := range objs {
				go func(obj *models.Object) {
					db.Transaction(func(tx *gorm.DB) error {
						defer wg.Done()
						// ctx.Logger.Info("rerunning object hooks", "id", obj.ID, "type", obj.Type, "uri", obj.URI)
						if err := obj.AfterCreate(tx); err != nil {
							ctx.Logger.Error("failed to run object hooks", "id", obj.ID, "type", obj.Type, "uri", obj.URI, "error", err)
						}
						return nil
					})
				}(obj)
			}
			wg.Wait()
			return nil
		}).Error
}
