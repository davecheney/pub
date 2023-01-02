package models

import "github.com/davecheney/pub/internal/snowflake"

type Attachment struct {
	snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	MediaType    string `gorm:"size:64;not null"`
	URL          string `gorm:"size:255;not null"`
	Name         string `gorm:"not null"`
	Blurhash     string `gorm:"size:36;not null"`
	Width        int    `gorm:"not null"`
	Height       int    `gorm:"not null"`
}

// A StatusAttachment is an attachment to a Status.
// A Status has many StatusAttachments.
type StatusAttachment struct {
	Attachment
	StatusID snowflake.ID `gorm:"not null"`
}
