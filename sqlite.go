//go:build sqlite

package main

// sqlite support

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDialector(dsn string) gorm.Dialector {
	return &sqlite.Dialector{
		DSN: dsn,
	}
}

func configureDB(db *gorm.DB) error {
	// enable foreign key constraints
	return db.Exec("PRAGMA foreign_keys = ON").Error
}
