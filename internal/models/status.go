package models

import (
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"gorm.io/gorm"
)

// A Status is a single message posted by a user. It may be a reply to another
// status, or a new thread of conversation.
// A Status belongs to a single Account, and is part of a single Conversation.
type Status struct {
	ID               uint64 `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt        time.Time
	ActorID          uint64
	Actor            *Actor
	ConversationID   uint32
	Conversation     *Conversation
	InReplyToID      *uint64
	InReplyToActorID *uint64
	Sensitive        bool
	SpoilerText      string `gorm:"size:128"`
	Visibility       string `gorm:"type:enum('public', 'unlisted', 'private', 'direct', 'limited')"`
	Language         string `gorm:"size:2"`
	Note             string
	URI              string `gorm:"uniqueIndex;size:128"`
	RepliesCount     int    `gorm:"not null;default:0"`
	ReblogsCount     int    `gorm:"not null;default:0"`
	FavouritesCount  int    `gorm:"not null;default:0"`
	ReblogID         *uint64
	Reblog           *Status
	Reaction         *Reaction
	Attachments      []StatusAttachment
}

func (st *Status) AfterCreate(tx *gorm.DB) error {
	return withTX(tx, st.updateStatusCount, st.updateRepliesCount)
}

// updateRepliesCount updates the replies_count field on the status.
func (st *Status) updateRepliesCount(tx *gorm.DB) error {
	if st.InReplyToID == nil {
		return nil
	}

	parent := &Status{ID: *st.InReplyToID}
	repliesCount := tx.Select("COUNT(id)").Where("in_reply_to_id = ?", *st.InReplyToID).Table("statuses")
	return tx.Model(parent).Update("replies_count", repliesCount).Error
}

// updateStatusCount updates the status_count and last_status_at fields on the actor.
func (st *Status) updateStatusCount(tx *gorm.DB) error {
	statusesCount := tx.Select("COUNT(id)").Where("actor_id = ?", st.ActorID).Table("statuses")
	createdAt := snowflake.ID(st.ID).IDToTime()
	return tx.Model(st.Actor).Updates(map[string]interface{}{
		"statuses_count": statusesCount,
		"last_status_at": createdAt,
	}).Error
}

type StatusPoll struct {
	ID         uint64 `gorm:"primarykey"`
	ExpiresAt  time.Time
	Multiple   bool
	VotesCount int                `gorm:"not null;default:0"`
	Options    []StatusPollOption `gorm:"serializer:json"`
}

type StatusPollOption struct {
	Title string `json:"title"`
	Count int    `json:"count"`
}
