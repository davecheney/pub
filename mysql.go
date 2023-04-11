//go:build !sqlite

package main

// mysql support

import (
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newDialector(dsn string) gorm.Dialector {
	return mysql.New(mysql.Config{
		DSN:                       mergeOptions(dsn, "charset=utf8mb4&parseTime=True&loc=Local"),
		SkipInitializeWithVersion: false, // auto configure based on currently MySQL version

	})
}

// merge options appends the options to the DSN if they are not already present.
func mergeOptions(dsn, options string) string {
	if options == "" {
		return dsn
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&" + options
	}
	return dsn + "?" + options
}

func configureDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	return nil
}
