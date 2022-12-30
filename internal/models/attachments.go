package models

import "github.com/davecheney/m/internal/snowflake"

type Attachment struct {
	snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	MediaType    string
	URL          string
	Name         string
	Blurhash     string
	Width        int
	Height       int
}

// A StatusAttachment is an attachment to a Status.
// A Status has many StatusAttachments.
type StatusAttachment struct {
	Attachment
	StatusID snowflake.ID
}
