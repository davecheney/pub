//go:build !sqlite

package models

type StatusVisibility struct {
	Visibility string `gorm:"type:enum('public', 'unlisted', 'private', 'direct', 'limited')"`
}
