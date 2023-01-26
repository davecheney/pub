package models

import (
	"gorm.io/gorm"
)

type Env struct {
	// DB is the database connection.
	DB *gorm.DB
}
