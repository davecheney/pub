package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// A Status is a single message posted by a user. It may be a reply to another
// status, or a new thread of conversation.
// A Status belongs to a single Account, and is part of a single Conversation.
type Status struct {
	ObjectID         snowflake.ID  `gorm:"primarykey;autoIncrement:false"`
	Object           *StatusObject `gorm:"constraint:OnDelete:CASCADE;<-:false"`
	UpdatedAt        time.Time     `gorm:"autoUpdateTime:false"`
	ActorID          snowflake.ID
	Actor            *Actor `gorm:"constraint:OnDelete:CASCADE;<-:false;"` // don't update actor on status update
	ConversationID   uint32
	Conversation     *Conversation `gorm:"constraint:OnDelete:CASCADE;"`
	InReplyToID      *snowflake.ID
	InReplyTo        *Status `gorm:"constraint:OnDelete:SET NULL;<-:false;"` // don't update in_reply_to on status update
	InReplyToActorID *snowflake.ID
	Visibility       Visibility `gorm:"not null"`
	RepliesCount     int        `gorm:"not null;default:0"`
	ReblogsCount     int        `gorm:"not null;default:0"`
	FavouritesCount  int        `gorm:"not null;default:0"`
	ReblogID         *snowflake.ID
	Reblog           *Status   `gorm:"constraint:OnDelete:CASCADE;<-:false;"` // don't update reblog on status update
	Reaction         *Reaction `gorm:"constraint:OnDelete:CASCADE;<-:false;"` // don't update reaction on status update
}

type StatusObject struct {
	ID         snowflake.ID
	Type       string
	URI        string
	Properties struct {
		Type string `json:"type"`
		// The Actor's unique global identifier.
		ID         string             `json:"id"`
		Content    string             `json:"content"`
		Sensitive  bool               `json:"sensitive"` // as:sensitive
		Summary    string             `json:"summary"`
		Attachment []StatusAttachment `json:"attachment"`
		Tag        []StatusTag        `json:"tag"`
	} `gorm:"serializer:json;not null"`
}

type StatusAttachment struct {
	Type       string `json:"type"`
	MediaType  string `json:"mediaType"`
	URL        string `json:"url"`
	Name       string `json:"name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Blurhash   string `json:"blurhash"`
	FocalPoint any    `json:"focalPoint"`
}

func (StatusObject) TableName() string {
	return "objects"
}

func (st *Status) AfterCreate(tx *gorm.DB) error {
	return forEach(tx,
		st.updateStatusCount,
		st.updateRepliesCount,
		st.updateReblogsCount,
	)
}

func (st *Status) Attachments() []*Attachment {
	return algorithms.Map(st.Object.Properties.Attachment, func(a StatusAttachment) *Attachment {
		return &Attachment{
			MediaType: a.MediaType,
			URL:       a.URL,
			Name:      a.Name,
			Width:     a.Width,
			Height:    a.Height,
			Blurhash:  a.Blurhash,
			// FocalPoint: FocalPoint{
			// 	X: func() float64 {
			// 		if len(a.FocalPoint) == 0 {
			// 			return 0
			// 		}
			// 		return a.FocalPoint[0]
			// 	}(),
			// 	Y: func() float64 {
			// 		if len(a.FocalPoint) < 2 {
			// 			return 0
			// 		}
			// 		return a.FocalPoint[1]
			// 	}(),
			// },
		}
	})

}

func (st *Status) Language() string {
	return "en" // todo
}

func (st *Status) Note() string {
	return st.Object.Properties.Content
}

func (st *Status) Sensitive() bool {
	return st.Object.Properties.Sensitive
}

func (st *Status) SpoilerText() string {
	return st.Object.Properties.Summary
}

func (st *Status) Tag() []StatusTag {
	return st.Object.Properties.Tag
}

func (st *Status) URI() string {
	return st.Object.Properties.ID
}

func (st *Status) AfterUpdate(tx *gorm.DB) error {
	return forEach(tx, st.updateStatusCount, st.updateRepliesCount, st.updateReblogsCount, st.maybeScheduleActorRefresh)
}

// updateRepliesCount updates the replies_count field on the status.
func (st *Status) updateRepliesCount(tx *gorm.DB) error {
	if st.InReplyToID == nil {
		return nil
	}

	parent := &Status{ObjectID: *st.InReplyToID}
	repliesCount := tx.Select("COUNT(*)").Where("in_reply_to_id = ?", *st.InReplyToID).Table("statuses")
	return tx.Model(parent).UpdateColumns(map[string]interface{}{
		"replies_count": repliesCount,
	}).Error
}

// updateReblogsCount updates the reblogs_count field on the status it reblogs.
func (st *Status) updateReblogsCount(tx *gorm.DB) error {
	if st.ReblogID == nil {
		return nil
	}

	reblog := &Status{ObjectID: *st.ReblogID}
	reblogsCount := tx.Select("COUNT(*)").Where("reblog_id = ?", *st.ReblogID).Table("statuses")
	return tx.Model(reblog).UpdateColumns(map[string]interface{}{
		"reblogs_count": reblogsCount,
	}).Error
}

// updateStatusCount updates the status_count and last_status_at fields on the actor.
func (st *Status) updateStatusCount(tx *gorm.DB) error {
	statusesCount := tx.Select("COUNT(*)").Where("actor_id = ?", st.ActorID).Table("statuses")
	createdAt := st.ObjectID.ToTime()
	actor := &Actor{ObjectID: st.ActorID}
	//TODO(dfc) last_status_at should only be updated if the status is newer than the current value.
	return tx.Model(actor).UpdateColumns(map[string]interface{}{
		"statuses_count": statusesCount,
		"last_status_at": createdAt,
	}).Error
}

func (st *Status) maybeScheduleActorRefresh(tx *gorm.DB) error {
	if st.Actor == nil {
		return fmt.Errorf("status %d has no actor", st.ObjectID)
	}
	if st.Actor.UpdatedAt.Before(time.Now().Add(-24 * time.Hour)) {
		return nil
	}
	fmt.Println("scheduling actor refresh:", st.Actor.ObjectID)
	return NewActors(tx).Refresh(st.Actor)
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
	Type string `json:"type"`
	Name string `json:"name"`
	HRef string `json:"href"`
}

// type StatusTag struct {
// 	StatusID snowflake.ID `gorm:"primarykey;autoIncrement:false"`
// 	TagID    uint32       `gorm:"primarykey;autoIncrement:false"`
// 	Tag      *Tag
// }

type Statuses struct {
	db *gorm.DB
}

func NewStatuses(db *gorm.DB) *Statuses {
	return &Statuses{db: db}
}

func (s *Statuses) FindOrCreateByURI(uri string) (*Status, error) {
	status, err := s.FindByURI(uri)
	if err == nil {
		// found
		return status, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// something went wrong
		return nil, err
	}
	// not found, create
	props, err := fetchObject(s.db.Statement.Context, uri)
	if err != nil {
		return nil, err
	}
	obj := &Object{
		Properties: props,
	}
	if err := s.db.
		Clauses(
			clause.Returning{
				Columns: []clause.Column{{Name: "id"}},
			},
			clause.OnConflict{
				Columns: []clause.Column{{Name: "uri"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"type",
					"properties",
				}),
			}).
		Save(obj).Error; err != nil {
		return nil, err
	}
	return s.FindByURI(uri)
}

func (s *Statuses) FindByURI(uri string) (*Status, error) {
	var status []Status
	if err := s.db.Joins("JOIN objects on objects.id = statuses.object_id").Preload("Conversation").Scopes(PreloadStatus).Where("objects.uri = ?", uri).Find(&status).Error; err != nil {
		return nil, err
	}
	if len(status) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &status[0], nil
}

func (s *Statuses) FindByID(id snowflake.ID) (*Status, error) {
	var status Status
	err := s.db.Preload("Conversation").Scopes(PreloadStatus).Take(&status, "statuses.object_id = ?", id).Error
	return &status, err
}

func (s *Statuses) Create(actor *Actor, parent *Status, visibility Visibility, sensitive bool, spoilterText, language, note string) (*Status, error) {
	createdAt := time.Now()
	conv := conversation(parent, visibility)
	id := snowflake.TimeToID(createdAt)
	status := Status{
		ObjectID:     id,
		UpdatedAt:    createdAt,
		ActorID:      actor.ObjectID,
		Actor:        actor,
		Conversation: conv,
		InReplyToID: func() *snowflake.ID {
			if parent != nil {
				return &parent.ObjectID
			}
			return nil
		}(),
		InReplyToActorID: func() *snowflake.ID {
			if parent != nil {
				return &parent.ActorID
			}
			return nil
		}(),
		// URI:         fmt.Sprintf("https://%s/users/%s/%d", actor.Domain, actor.Name, id),
		// Sensitive:   sensitive,
		// SpoilerText: spoilterText,
		Visibility: visibility,
		// Language:    language,
		// Note:        note,
	}
	return &status, s.db.Create(&status).Error
}

// conversation returns the conversation of the parent, or a new conversation if parent is nil.
func conversation(parent *Status, visibility Visibility) *Conversation {
	if parent != nil {
		return parent.Conversation
	}
	return &Conversation{
		Visibility: visibility,
	}
}

// PreloadStatus preloads all of a Status' relations and associations.
func PreloadStatus(query *gorm.DB) *gorm.DB {
	// return query.Preload("Attachments").
	// Preload("Poll").Preload("Poll.Options").
	// Preload("Mentions").Preload("Mentions.Actor").Preload("Mentions.Actor.Object").
	return query.Preload("Object").Preload("Actor").Preload("Actor.Object").
		// Preload("Tags").Preload("Tags.Tag").
		Preload("Reblog").Preload("Reblog.Object").
		Preload("Reblog.Actor").Preload("Reblog.Actor.Object")
	// Preload("Reblog.Attachments").
	// Preload("Reblog.Poll").Preload("Reblog.Poll.Options").
	// Preload("Reblog.Mentions").Preload("Reblog.Mentions.Actor").Preload("Reblog.Mentions.Actor.Object").
	// Preload("Reblog.Tags").Preload("Reblog.Tags.Tag")
}

// PreloadReaction preloads all of a Reaction's relations and associations.
func PreloadReaction(actor *Actor) func(query *gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		return query.Preload("Reaction", "actor_id = ?", actor.ObjectID).Preload("Reblog.Reaction", "actor_id = ?", actor.ObjectID)
	}
}
