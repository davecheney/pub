package models

import (
	"fmt"
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Reaction represents an an actors reaction to a status.
type Reaction struct {
	StatusID   snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Status     *Status      `gorm:"constraint:OnDelete:CASCADE;"`
	ActorID    snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Actor      *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
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
	status := &Status{ID: snowflake.ID(r.StatusID)}
	favouritesCount := tx.Select("COUNT(*)").Where("status_id = ? and favourited = true", r.StatusID).Table("reactions")
	reblogsCount := tx.Select("COUNT(*)").Where("status_id = ? and reblogged = true", r.StatusID).Table("reactions")
	return tx.Model(status).Updates(map[string]interface{}{
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
	ActorID   snowflake.ID `gorm:"uniqueIndex:idx_actor_id_target_id;not null;"`
	// Actor is the actor that is requesting the reaction change.
	Actor    *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
	TargetID snowflake.ID `gorm:"uniqueIndex:idx_actor_id_target_id;not null;"`
	// Target is the status that is being reacted to.
	Target *Status `gorm:"constraint:OnDelete:CASCADE;"`
	// Action is the action to perform, either follow or unfollow.
	Action string `gorm:"type:enum('like', 'unlike');not null"`
	// Attempts is the number of times the request has been attempted.
	Attempts uint32 `gorm:"not null;default:0"`
	// LastAttempt is the time the request was last attempted.
	LastAttempt time.Time
	// LastResult is the result of the last attempt if it failed.
	LastResult string `gorm:"size:255;not null;default:''"`
}
