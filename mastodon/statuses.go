package mastodon

import (
	"gorm.io/gorm"
)

type Status struct {
	gorm.Model
	AccountID uint
	Account   Account
}
