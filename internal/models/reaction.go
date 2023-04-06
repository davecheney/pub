package models

import (
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Reaction represents an an actors reaction to a status.
type Reaction struct {
	StatusID   snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Status     *Status      `gorm:"constraint:OnDelete:CASCADE;<-:false"`
	ActorID    snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Actor      *Actor       `gorm:"constraint:OnDelete:CASCADE;<-:false"`
	Favourited bool         `gorm:"not null;default:false"`
	Reblogged  bool         `gorm:"not null;default:false"`
	Muted      bool         `gorm:"not null;default:false"`
	Bookmarked bool         `gorm:"not null;default:false"`
	Pinned     bool         `gorm:"not null;default:false"`
}

// BeforeUpdate creates a reaction request between the actor and target if needed.
func (r *Reaction) BeforeUpdate(tx *gorm.DB) error {
	var original Reaction
	if err := tx.First(&original, "actor_id = ? and status_id = ?", r.ActorID, r.StatusID).Error; err != nil {
		return err
	}
	fmt.Printf("reaction changed from %+v to %+v\n", original, r)

	// if there is a conflict; eg. a follow then an unfollow before the follow is processed
	// update the existing row to reflect the new action.
	tx = tx.Clauses(clause.OnConflict{
		UpdateAll: true,
	})

	// what changed?
	switch {
	case original.Favourited && !r.Favourited:
		// undo like
		return tx.Create(&ReactionRequest{
			ActorID:  r.ActorID,
			TargetID: r.StatusID,
			Action:   "unlike",
		}).Error
	case !original.Favourited && r.Favourited:
		// like
		return tx.Create(&ReactionRequest{
			ActorID:  r.ActorID,
			TargetID: r.StatusID,
			Action:   "like",
		}).Error
	default:
		return nil
	}
}

func (r *Reaction) AfterUpdate(tx *gorm.DB) error {
	return forEach(tx, r.updateStatusCount)
}

// updateStatusCount updates the favourites_count and reblogs_count fields on the status.
func (r *Reaction) updateStatusCount(tx *gorm.DB) error {
	status := &Status{ID: r.StatusID}
	favouritesCount := tx.Select("COUNT(*)").Where("status_id = ? and favourited = true", r.StatusID).Table("reactions")
	reblogsCount := tx.Select("COUNT(*)").Where("status_id = ? and reblogged = true", r.StatusID).Table("reactions")
	return tx.Model(status).UpdateColumns(map[string]interface{}{
		"favourites_count": favouritesCount,
		"reblogs_count":    reblogsCount,
	}).Error
}

// A ReactionRequest is a request to update the reaction to a status.
// ReactionRequests are created by hooks on the Reaction model, and are
// processed by the ReactionRequestProcessor in the background.
type ReactionRequest struct {
	ID uint32 `gorm:"primarykey;"`
	// CreatedAt is the time the request was created.
	CreatedAt time.Time
	// UpdatedAt is the time the request was last updated.
	UpdatedAt time.Time
	ActorID   snowflake.ID `gorm:"uniqueIndex:uidx_reaction_requests_actor_id_target_id;not null;"`
	// Actor is the actor that is requesting the reaction change.
	Actor    *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
	TargetID snowflake.ID `gorm:"uniqueIndex:uidx_reaction_requests_actor_id_target_id;not null;"`
	// Target is the status that is being reacted to.
	Target *Status `gorm:"constraint:OnDelete:CASCADE;"`
	// Action is the action to perform, either follow or unfollow.
	Action ReactionRequestAction `gorm:"not null"`
	// Attempts is the number of times the request has been attempted.
	Attempts uint32 `gorm:"not null;default:0"`
	// LastAttempt is the time the request was last attempted.
	LastAttempt time.Time
	// LastResult is the result of the last attempt if it failed.
	LastResult string `gorm:"size:255;not null;default:''"`
}

type ReactionRequestAction string

func (ReactionRequestAction) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "postgres":
		return "enum('like', 'unlike')"
	case "sqlite":
		return "TEXT"
	default:
		return ""
	}
}

type Reactions struct {
	db *gorm.DB
}

func NewReactions(db *gorm.DB) *Reactions {
	return &Reactions{db: db}
}

func (r *Reactions) Pin(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Pinned = true
	err = r.db.Model(reaction).UpdateColumn("pinned", true).Error
	return reaction, err
}

func (r *Reactions) Unpin(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Pinned = false
	err = r.db.Model(reaction).UpdateColumn("pinned", false).Error
	return reaction, err
}

func (r *Reactions) Favourite(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Favourited = true
	if err := r.db.Model(reaction).UpdateColumn("favourited", true).Error; err != nil {
		return nil, err
	}
	reaction.Status.FavouritesCount++
	return reaction, nil
}

func (r *Reactions) Unfavourite(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Favourited = false
	if err := r.db.Model(reaction).UpdateColumn("favourited", false).Error; err != nil {
		return nil, err
	}
	reaction.Status.FavouritesCount--
	return reaction, nil
}

func (r *Reactions) Bookmark(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Bookmarked = true
	err = r.db.Model(reaction).Update("bookmarked", true).Error
	return reaction, err
}

func (r *Reactions) Unbookmark(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Bookmarked = false
	err = r.db.Model(reaction).Update("bookmarked", false).Error
	return reaction, err
}

// Reblog creates a new status that is a reblog of the given status.
func (r *Reactions) Reblog(status *Status, actor *Actor) (*Status, error) {
	return withTransaction(r.db, func(tx *gorm.DB) (*Status, error) {
		conv := Conversation{
			Visibility: "public",
		}
		if err := r.db.Create(&conv).Error; err != nil {
			return nil, err
		}

		reaction, err := r.findOrCreate(status, actor)
		if err != nil {
			return nil, err
		}
		reaction.Reblogged = true
		err = r.db.Model(reaction).Update("reblogged", true).Error
		if err != nil {
			return nil, err
		}

		id := snowflake.Now()
		reblog := Status{
			ID:             id,
			ActorID:        actor.ID,
			Actor:          actor,
			ConversationID: conv.ID,
			Visibility:     conv.Visibility,
			ReblogID:       &status.ID,
			Reblog:         status,
			URI:            fmt.Sprintf("%s/statuses/%d", actor.URI, id),
			Reaction:       reaction,
		}
		if err := r.db.Create(&reblog).Error; err != nil {
			return nil, err
		}
		return &reblog, nil
	})
}

// Unreblog removes the reblog of the given status with the given actor.
func (r *Reactions) Unreblog(status *Status, actor *Actor) (*Status, error) {
	return withTransaction(r.db, func(tx *gorm.DB) (*Status, error) {
		reaction, err := r.findOrCreate(status, actor)
		if err != nil {
			return nil, err
		}
		reaction.Reblogged = false
		err = r.db.Model(reaction).Update("reblogged", false).Error
		if err != nil {
			return nil, err
		}
		var reblog Status
		if err := r.db.Where("reblog_id = ? AND actor_id = ?", status.ID, actor.ID).First(&reblog).Error; err != nil {
			return nil, err
		}
		if err := r.db.Delete(&reblog).Error; err != nil {
			return nil, err
		}
		return status, nil
	})
}

func (r *Reactions) findOrCreate(status *Status, actor *Actor) (*Reaction, error) {
	var reaction Reaction
	if err := r.db.FirstOrCreate(&reaction, Reaction{StatusID: status.ID, ActorID: actor.ID}).Error; err != nil {
		return nil, err
	}
	status.Reaction = &reaction
	reaction.Status = status
	reaction.Actor = actor
	return &reaction, nil
}
