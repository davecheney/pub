//go:build sqlite

package models

type TokenType struct {
	Type string `gorm:"not null"`
}
