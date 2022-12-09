package m

import (
	"gorm.io/gorm"
)

type Notification struct {
	gorm.Model
	AccountID uint
	Account   *Account
	StatusID  *uint
	Status    *Status
	Type      string `gorm:"size:64"`
}
