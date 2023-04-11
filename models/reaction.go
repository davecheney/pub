package models

import (
	"fmt"

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

func (r *Reaction) BeforeUpdate(tx *gorm.DB) error {
	return forEach(tx, r.createReactionRequest)
}

func (r *Reaction) AfterSave(tx *gorm.DB) error {
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

// createReactionRequest creates a reaction request between the actor and target if needed.
func (r *Reaction) createReactionRequest(tx *gorm.DB) error {
	var original Reaction
	if err := tx.First(&original, "actor_id = ? and status_id = ?", r.ActorID, r.StatusID).Error; err != nil {
		return err
	}
	fmt.Printf("reaction changed from %+v to %+v\n", original, r)

	// if there is a conflict; eg. a like then an unlike before the follow is processed
	// update the existing row to reflect the new action.
	tx = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "actor_id"}, {Name: "target_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"action",
			"created_at",
			"updated_at",
			"attempts", // resets the attempts counter
		}),
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

// A ReactionRequest is a request to update the reaction to a status.
// ReactionRequests are created by hooks on the Reaction model, and are
// processed by the ReactionRequestProcessor in the background.
type ReactionRequest struct {
	Request

	// ActorID is the ID of the actor that is requesting the reaction change.
	ActorID snowflake.ID `gorm:"uniqueIndex:uidx_reaction_requests_actor_id_target_id;not null;"`
	// Actor is the actor that is requesting the reaction change.
	Actor    *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
	TargetID snowflake.ID `gorm:"uniqueIndex:uidx_reaction_requests_actor_id_target_id;not null;"`
	// Target is the status that is being reacted to.
	Target *Status `gorm:"constraint:OnDelete:CASCADE;"`
	// Action is the action to perform, either follow or unfollow.
	Action ReactionRequestAction `gorm:"not null"`
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
	reaction, err := findOrCreateReaction(r.db, status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Pinned = true
	return reaction, r.db.Save(reaction).Error
}

func (r *Reactions) Unpin(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := findOrCreateReaction(r.db, status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Pinned = false
	return reaction, r.db.Save(reaction).Error
}

func (r *Reactions) Favourite(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := findOrCreateReaction(r.db, status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Favourited = true
	reaction.Status.FavouritesCount++
	return reaction, r.db.Save(reaction).Error
}

func (r *Reactions) Unfavourite(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := findOrCreateReaction(r.db, status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Favourited = false
	reaction.Status.FavouritesCount--
	return reaction, r.db.Save(reaction).Error
}

func (r *Reactions) Bookmark(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := findOrCreateReaction(r.db, status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Bookmarked = true
	return reaction, r.db.Save(reaction).Error
}

func (r *Reactions) Unbookmark(status *Status, actor *Actor) (*Reaction, error) {
	reaction, err := findOrCreateReaction(r.db, status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Bookmarked = false
	return reaction, r.db.Save(reaction).Error
}

// Reblog creates a new status that is a reblog of the given status.
func (r *Reactions) Reblog(status *Status, actor *Actor) (*Status, error) {
	var reblog Status
	return &reblog, r.db.Transaction(func(tx *gorm.DB) error {
		reaction, err := findOrCreateReaction(tx, status, actor)
		if err != nil {
			return err
		}
		reaction.Reblogged = true
		reaction.Status.ReblogsCount++
		if err := tx.Save(reaction).Error; err != nil {
			return err
		}

		id := snowflake.Now()
		reblog = Status{
			ID:      id,
			ActorID: actor.ID,
			Actor:   actor,
			Conversation: &Conversation{
				Visibility: "public",
			},
			Visibility: "public",
			ReblogID:   &status.ID,
			Reblog:     status,
			URI:        fmt.Sprintf("%s/statuses/%d", actor.URI, id),
			Reaction:   reaction,
		}
		return tx.Create(&reblog).Error
	})
}

// Unreblog removes the reblog of the given status with the given actor.
func (r *Reactions) Unreblog(status *Status, actor *Actor) (*Status, error) {
	var reblog Status
	return &reblog, r.db.Transaction(func(tx *gorm.DB) error {
		reaction, err := findOrCreateReaction(tx, status, actor)
		if err != nil {
			return err
		}
		reaction.Reblogged = false
		reaction.Status.ReblogsCount--
		if err := tx.Save(reaction).Error; err != nil {
			return err
		}

		if err := tx.Where("reblog_id = ? AND actor_id = ?", status.ID, actor.ID).Preload("Actor").First(&reblog).Error; err != nil {
			return err
		}
		return tx.Delete(&reblog).Error
	})
}

func findOrCreateReaction(tx *gorm.DB, status *Status, actor *Actor) (*Reaction, error) {
	status.Reaction = &Reaction{
		StatusID: status.ID,
		Status:   status,
		ActorID:  actor.ID,
		Actor:    actor,
	}
	return status.Reaction, tx.FirstOrCreate(status.Reaction).Error
}
