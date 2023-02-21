//go:build sqlite

package models

type ReactionRequestAction struct {
	Action string `gorm:not null"`
}
