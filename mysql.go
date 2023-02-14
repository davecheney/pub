//go:build !sqlite

package main

// mysql support

import (
	"strings"

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
