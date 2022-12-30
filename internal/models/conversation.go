package models

import (
	"time"
)

// A Conversation is a collection of related statuses. It is a way to group
// together statuses that are replies to each other, or that are part of the
// same thread of conversation. Conversations are not necessarily public, and
// may be limited to a set of participants.
type Conversation struct {
	ID         uint32 `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Visibility string `gorm:"type:enum('public', 'unlisted', 'private', 'direct', 'limited');not null"`
}
