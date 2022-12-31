package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"gorm.io/gorm"
)

// A Status is a single message posted by a user. It may be a reply to another
// status, or a new thread of conversation.
// A Status belongs to a single Account, and is part of a single Conversation.
type Status struct {
	snowflake.ID     `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt        time.Time
	ActorID          snowflake.ID
	Actor            *Actor `gorm:"constraint:OnDelete:CASCADE;"`
	ConversationID   uint32
	Conversation     *Conversation `gorm:"constraint:OnDelete:CASCADE;"`
	InReplyToID      *snowflake.ID
	InReplyToActorID *snowflake.ID
	Sensitive        bool
	SpoilerText      string `gorm:"size:128"`
	Visibility       string `gorm:"type:enum('public', 'unlisted', 'private', 'direct', 'limited')"`
	Language         string `gorm:"size:2"`
	Note             string
	URI              string `gorm:"uniqueIndex;size:128"`
	RepliesCount     int    `gorm:"not null;default:0"`
	ReblogsCount     int    `gorm:"not null;default:0"`
	FavouritesCount  int    `gorm:"not null;default:0"`
	ReblogID         *snowflake.ID
	Reblog           *Status
	Reaction         *Reaction
	Attachments      []StatusAttachment `gorm:"constraint:OnDelete:CASCADE;"`
}

func (st *Status) AfterCreate(tx *gorm.DB) error {
	return forEach(tx, st.updateStatusCount, st.updateRepliesCount)
}

// updateRepliesCount updates the replies_count field on the status.
func (st *Status) updateRepliesCount(tx *gorm.DB) error {
	if st.InReplyToID == nil {
		return nil
	}

	parent := &Status{ID: snowflake.ID(*st.InReplyToID)}
	repliesCount := tx.Select("COUNT(id)").Where("in_reply_to_id = ?", *st.InReplyToID).Table("statuses")
	return tx.Model(parent).Update("replies_count", repliesCount).Error
}

// updateStatusCount updates the status_count and last_status_at fields on the actor.
func (st *Status) updateStatusCount(tx *gorm.DB) error {
	statusesCount := tx.Select("COUNT(id)").Where("actor_id = ?", st.ActorID).Table("statuses")
	createdAt := st.ID.ToTime()
	return tx.Model(&Actor{
		ID: st.ActorID,
	}).Updates(map[string]interface{}{
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

type Statuses struct {
	db *gorm.DB
}

func NewStatuses(db *gorm.DB) *Statuses {
	return &Statuses{db: db}
}

// FindOrCreate searches for a status by its URI. If the status is not found, it
// calls the given function to create a new status, stores that status in the
// database and returns it.
func (s *Statuses) FindOrCreate(uri string, createFn func(string) (*Status, error)) (*Status, error) {
	status, err := s.FindByURI(uri)
	if err == nil {
		return status, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	status, err = createFn(uri)
	if err != nil {
		fmt.Println("findOrCreate: createFn:", err)
		return nil, err
	}
	if err := s.db.Create(&status).Error; err != nil {
		return nil, err
	}
	return status, nil
}

func (s *Statuses) FindByURI(uri string) (*Status, error) {
	var status Status
	if err := s.db.Preload("Actor").Where("uri = ?", uri).First(&status).Error; err != nil {
		return nil, err
	}
	return &status, nil
}
