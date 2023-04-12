package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// A Status is a single message posted by a user. It may be a reply to another
// status, or a new thread of conversation.
// A Status belongs to a single Account, and is part of a single Conversation.
type Status struct {
	snowflake.ID     `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime:false"`
	ActorID          snowflake.ID
	Actor            *Actor `gorm:"constraint:OnDelete:CASCADE;<-:false;"` // don't update actor on status update
	ConversationID   uint32
	Conversation     *Conversation `gorm:"constraint:OnDelete:CASCADE;"`
	InReplyToID      *snowflake.ID
	InReplyToActorID *snowflake.ID
	Sensitive        bool
	SpoilerText      string     `gorm:"size:128"`
	Visibility       Visibility `gorm:"not null"`
	Language         string     `gorm:"size:2"`
	Note             string     `gorm:"type:text"`
	URI              string     `gorm:"uniqueIndex;size:128"`
	RepliesCount     int        `gorm:"not null;default:0"`
	ReblogsCount     int        `gorm:"not null;default:0"`
	FavouritesCount  int        `gorm:"not null;default:0"`
	ReblogID         *snowflake.ID
	Reblog           *Status             `gorm:"constraint:OnDelete:CASCADE;<-:false;"` // don't update reblog on status update
	Reaction         *Reaction           `gorm:"constraint:OnDelete:CASCADE;<-:false;"` // don't update reaction on status update
	Attachments      []*StatusAttachment `gorm:"constraint:OnDelete:CASCADE;"`
	Mentions         []StatusMention     `gorm:"constraint:OnDelete:CASCADE;"`
	Tags             []StatusTag         `gorm:"constraint:OnDelete:CASCADE;"`
	Poll             *StatusPoll         `gorm:"constraint:OnDelete:CASCADE;"`
}

func (st *Status) AfterCreate(tx *gorm.DB) error {
	return forEach(tx,
		st.updateStatusCount,
		st.updateRepliesCount,
		st.updateReblogsCount,
	)
}

func (st *Status) AfterUpdate(tx *gorm.DB) error {
	return forEach(tx, st.updateStatusCount, st.updateRepliesCount, st.updateReblogsCount)
}

// updateRepliesCount updates the replies_count field on the status.
func (st *Status) updateRepliesCount(tx *gorm.DB) error {
	if st.InReplyToID == nil {
		return nil
	}

	parent := &Status{ID: *st.InReplyToID}
	repliesCount := tx.Select("COUNT(id)").Where("in_reply_to_id = ?", *st.InReplyToID).Table("statuses")
	return tx.Model(parent).UpdateColumns(map[string]interface{}{
		"replies_count": repliesCount,
	}).Error
}

// updateReblogsCount updates the reblogs_count field on the status it reblogs.
func (st *Status) updateReblogsCount(tx *gorm.DB) error {
	if st.ReblogID == nil {
		return nil
	}

	reblog := &Status{ID: *st.ReblogID}
	reblogsCount := tx.Select("COUNT(id)").Where("reblog_id = ?", *st.ReblogID).Table("statuses")
	return tx.Model(reblog).UpdateColumns(map[string]interface{}{
		"reblogs_count": reblogsCount,
	}).Error
}

// updateStatusCount updates the status_count and last_status_at fields on the actor.
func (st *Status) updateStatusCount(tx *gorm.DB) error {
	statusesCount := tx.Select("COUNT(id)").Where("actor_id = ?", st.ActorID).Table("statuses")
	createdAt := st.ID.ToTime()
	actor := &Actor{ID: st.ActorID}
	//TODO(dfc) last_status_at should only be updated if the status is newer than the current value.
	return tx.Model(actor).UpdateColumns(map[string]interface{}{
		"statuses_count": statusesCount,
		"last_status_at": createdAt,
	}).Error
}

// A Conversation is a collection of related statuses. It is a way to group
// together statuses that are replies to each other, or that are part of the
// same thread of conversation. Conversations are not necessarily public, and
// may be limited to a set of participants.
type Conversation struct {
	ID         uint32 `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Visibility Visibility `gorm:"not null;check <> ''"`
}

type Visibility string

func (Visibility) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "postgres":
		return "enum('public', 'unlisted', 'private', 'direct', 'limited')"
	case "sqlite":
		return "TEXT"
	default:
		return ""
	}
}

type StatusPoll struct {
	StatusID   snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	ExpiresAt  time.Time
	Multiple   bool
	VotesCount int                `gorm:"not null;default:0"`
	Options    []StatusPollOption `gorm:"constraint:OnDelete:CASCADE;"`
}

func (st *StatusPoll) AfterCreate(tx *gorm.DB) error {
	return forEach(tx, st.updateVotesCount)
}

func (st *StatusPoll) updateVotesCount(tx *gorm.DB) error {
	votesCount := tx.Select("SUM(count)").Where("status_poll_id = ?", st.StatusID).Table("status_poll_options")
	poll := &StatusPoll{StatusID: st.StatusID}
	return tx.Model(poll).UpdateColumns(map[string]interface{}{
		"votes_count": votesCount,
	}).Error
}

type StatusPollOption struct {
	ID           uint32 `gorm:"primarykey;autoIncrement:true"`
	StatusPollID snowflake.ID
	Title        string `gorm:"size:255;not null"`
	Count        int    `gorm:"not null;default:0"`
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
	if uri == "" {
		return nil, errors.New("Statuses.FindByURI: uri is empty")
	}
	// use find to avoid the not found error on empty result
	var status []Status
	query := s.db.Joins("Actor").Preload("Conversation").Scopes(PreloadStatus)
	if err := query.Where(&Status{URI: uri}).Find(&status).Error; err != nil {
		return nil, err
	}
	if len(status) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &status[0], nil
}

func (s *Statuses) FindByID(id snowflake.ID) (*Status, error) {
	var status Status
	query := s.db.Joins("Actor").Scopes(PreloadStatus)
	if err := query.First(&status, id).Error; err != nil {
		return nil, err
	}
	return &status, nil
}

// PreloadStatus preloads all of a Status' relations and associations.
func PreloadStatus(query *gorm.DB) *gorm.DB {
	return query.Preload("Attachments").
		Preload("Poll").Preload("Poll.Options").
		Preload("Mentions").Preload("Mentions.Actor").
		Preload("Tags").Preload("Tags.Tag").
		Preload("Reblog").
		Preload("Reblog.Actor").
		Preload("Reblog.Attachments").
		Preload("Reblog.Poll").Preload("Reblog.Poll.Options").
		Preload("Reblog.Mentions").Preload("Reblog.Mentions.Actor").
		Preload("Reblog.Tags").Preload("Reblog.Tags.Tag")
}

// PreloadReaction preloads all of a Reaction's relations and associations.
func PreloadReaction(actor *Actor) func(query *gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		return query.Preload("Reaction", "actor_id = ?", actor.ID).Preload("Reblog.Reaction", "actor_id = ?", actor.ID)
	}
}
