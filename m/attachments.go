package m

import "gorm.io/gorm"

type Attachment struct {
	gorm.Model
	Type     string
	URL      string
	Name     string
	Blurhash string
	Width    int
	Height   int
}

type StatusAttachment struct {
	Attachment
	StatusID uint64
}
