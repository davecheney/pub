//go:build !sqlite

package models

type ActorType struct {
	Type string `gorm:"type:enum('Person', 'Application', 'Service', 'Group', 'Organization', 'LocalPerson', 'LocalService');default:'Person';not null"`
}
