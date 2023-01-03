package models

import (
	"gorm.io/gorm"
)

type Env struct {
	// DB is the database connection.
	DB *gorm.DB
}

func (e *Env) Statuses() *Statuses {
	return &Statuses{
		db: e.DB,
	}
}
