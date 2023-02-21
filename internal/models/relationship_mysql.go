//go:build !sqlite

package models

type RelationshipRequestAction struct {
	Action string `gorm:"type:enum('follow', 'unfollow');not null"`
}
