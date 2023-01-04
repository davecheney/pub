package models

type Tag struct {
	ID   uint32 `gorm:"primaryKey"`
	Name string `gorm:"size:64;uniqueIndex"`
}
