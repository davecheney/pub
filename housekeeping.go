package main

import (
	"fmt"

	"gorm.io/gorm"
)

type HouseKeepingCmd struct {
}

func (c *HouseKeepingCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// delete all ActorAttributes that are not referenced by an Actor
		res := tx.Exec(`
			DELETE FROM actor_attributes
			WHERE actor_id NOT IN (SELECT id FROM actors)
		`)
		if res.Error != nil {
			return res.Error
		}
		fmt.Println("deleted", res.RowsAffected, "orphaned actor attributes")

		// delete all Person and Service actors that have no status
		res = tx.Exec(`
			DELETE FROM actors
			WHERE id IN (
				SELECT id FROM actors
				WHERE type IN ('Person', 'Service')
				AND id NOT IN (
					SELECT actor_id FROM statuses
				)
			)
		`)
		if res.Error != nil {
			return res.Error
		}
		fmt.Println("deleted", res.RowsAffected, "actors with no statuses")

		// delete all conversations that have no statuses
		res = tx.Exec(`
			DELETE FROM conversations
			WHERE id NOT IN (
				SELECT conversation_id FROM statuses
			)
		`)
		if res.Error != nil {
			return res.Error
		}
		fmt.Println("deleted", res.RowsAffected, "orphaned conversations")

		// delete all mentions that have no status or actor
		res = tx.Exec(`
			DELETE FROM status_mentions
			WHERE status_id NOT IN (
				SELECT id FROM statuses
			) OR actor_id NOT IN (
				SELECT id FROM actors
			)
		`)
		if res.Error != nil {
			return res.Error
		}
		fmt.Println("deleted", res.RowsAffected, "orphaned mentions")

		return nil
	})
}
