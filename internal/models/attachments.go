package models

import (
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

func (s *StatusAttachment) AfterSave(tx *gorm.DB) error {
	tx = tx.Clauses(clause.OnConflict{
		UpdateAll: true,
	})
	if s.URL == "" {
		// no URL, so no need to fetch the attachment
		return nil
	}
	if s.Height > 0 && s.Width > 0 {
		// already have the dimensions, so no need to fetch the attachment
		return nil
	}
	switch s.MediaType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		// supported media type, so fetch the attachment
		return tx.Create(&StatusAttachmentRequest{
			StatusAttachmentID: s.ID,
		}).Error
	default:
		// unsupported media type, so no need to fetch the attachment
		return nil
	}
}

// A StatusAttachmentRequest records a request fetch a remote attachment.
// StatusAttachmentRequest are created by hooks on the StatusAttachment model, and are
// processed by the StatusAttachmentRequestProcessor in the background.
type StatusAttachmentRequest struct {
	ID uint32 `gorm:"primarykey;"`
	// CreatedAt is the time the request was created.
	CreatedAt time.Time
	// UpdatedAt is the time the request was last updated.
	UpdatedAt time.Time
	// StatusAttachmentID is the ID of the StatusAttachment that the request is for.
	StatusAttachmentID snowflake.ID `gorm:"uniqueIndex;not null;"`
	// StatusAttachment is the StatusAttachment that the request is for.
	StatusAttachment *StatusAttachment `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
	// Attempts is the number of times the request has been attempted.
	Attempts uint32 `gorm:"not null;default:0"`
	// LastAttempt is the time the request was last attempted.
	LastAttempt time.Time
	// LastResult is the result of the last attempt if it failed.
	LastResult string `gorm:"size:255;not null;default:''"`
}
