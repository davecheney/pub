package models

type Attachment struct {
	ID        uint64 `gorm:"primarykey;autoIncrement:false"`
	MediaType string
	URL       string
	Name      string
	Blurhash  string
	Width     int
	Height    int
}

// A StatusAttachment is an attachment to a Status.
// A Status has many StatusAttachments.
type StatusAttachment struct {
	Attachment
	StatusID uint64
}
