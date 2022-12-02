package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/davecheney/m/m"
	"gorm.io/gorm"
)

type PurgeCmd struct {
	URI string `short:"u" long:"uri" description:"URI to purge"`
}

func (c *PurgeCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}
	tx := db.Begin()
	if isStatusURI(c.URI) {
		if err := purgeStatus(tx, c.URI); err != nil {
			tx.Rollback()
			return err
		}
		return tx.Commit().Error
	}
	return fmt.Errorf("unknown URI type: %s", c.URI)
}

func purgeStatus(tx *gorm.DB, uri string) error {
	var status m.Status
	if err := tx.Where("uri = ?", uri).First(&status).Error; err != nil {
		return err
	}
	if err := tx.Delete(&status).Error; err != nil {
		return err
	}
	return nil
}

func isStatusURI(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	parts := strings.Split(u.Path, "/")
	if len(parts) < 2 {
		return false
	}
	if parts[len(parts)-2] != "statuses" {
		return false
	}
	return true
}
