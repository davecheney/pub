package models

import (
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type Env struct {
	// DB is the database connection.
	DB     *gorm.DB
	Logger *slog.Logger
}

func (e *Env) Log() *slog.Logger {
	return e.Logger
}
