package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
)

// A Status is a single message posted by a user. It may be a reply to another
// status, or a new thread of conversation.
// A Status belongs to a single Account, and is part of a single Conversation.
type Status struct {
	snowflake.ID     `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt        time.Time
	ActorID          snowflake.ID
	Actor            *Actor `gorm:"constraint:OnDelete:CASCADE;<-:false;"` // don't update actor on status update
	ConversationID   uint32
	Conversation     *Conversation `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
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
	Reblog           *Status            `gorm:"<-:false;"` // don't update reblog on status update
	Reaction         *Reaction          `gorm:"<-:false;"` // don't update reaction on status update
	Attachments      []StatusAttachment `gorm:"constraint:OnDelete:CASCADE;"`
	Mentions         []StatusMention    `gorm:"constraint:OnDelete:CASCADE;"`
	Tags             []StatusTag        `gorm:"constraint:OnDelete:CASCADE;"`
}

func (st *Status) AfterCreate(tx *gorm.DB) error {
	return forEach(tx, st.updateStatusCount, st.updateRepliesCount)
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
	createdAt := st.ID.ToTime()
	actor := &Actor{ID: st.ActorID}
	return tx.Model(actor).Updates(map[string]interface{}{
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

type StatusMention struct {
	StatusID snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	ActorID  snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Actor    *Actor       `gorm:"constraint:OnDelete:CASCADE;<-:false;"` // don't update actor on mention update
}

type StatusTag struct {
	StatusID snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	TagID    uint32       `gorm:"primarykey;autoIncrement:false"`
	Tag      *Tag
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
		return nil, fmt.Errorf("Statuses.FindOrCreate: %w", err)
	}
	if err := s.db.Create(&status).Error; err != nil {
		return nil, err
	}
	return status, nil
}

func (s *Statuses) FindByURI(uri string) (*Status, error) {
	// use find to avoid the not found error on empty result
	var status []Status
	if err := s.db.Joins("Actor").Preload("Reblog").Preload("Reblog.Actor").Preload("Attachments").Where(&Status{URI: uri}).Find(&status).Error; err != nil {
		return nil, err
	}
	if len(status) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &status[0], nil
}
