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
