//go:build !sqlite

package models

type ConversationVisibility struct {
	Visibility string `gorm:"type:enum('public', 'unlisted', 'private', 'direct', 'limited');not null"`
}
