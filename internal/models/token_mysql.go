//go:build !sqlite

package models

type TokenType struct {
	Type string `gorm:"type:enum('Bearer');not null"`
}
