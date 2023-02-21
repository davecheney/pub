//go:build sqlite

package models

type ConversationVisibility struct {
	Visibility string `gorm:not null"`
}
