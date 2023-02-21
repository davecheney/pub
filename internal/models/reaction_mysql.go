//go:build !sqlite

package models

type ReactionRequestAction struct {
	Action string `gorm:"type:enum('like', 'unlike');not null"`
}
