//go:build sqlite

package models

type ActorType struct {
	Type string `gorm:default:'Person';not null"`
}
