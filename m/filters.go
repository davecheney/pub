package m

import (
	"time"

	"gorm.io/gorm"
)

// https://docs.joinmastodon.org/entities/V1_Filter/
type ClientFilter struct {
	gorm.Model
	AccountID    uint
	Account      *Account
	Phrase       string
	WholeWord    bool
	Context      []string `gorm:"serializer:json"`
	ExpiresAt    time.Time
	Irreversible bool
}
