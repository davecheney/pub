//go:build sqlite

package models

type RelationshipRequestAction struct {
	Action string `gorm:not null"`
}
