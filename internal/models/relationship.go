package models

import (
	"fmt"
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"gorm.io/gorm"
)

type Relationship struct {
	ActorID    snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Actor      *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
	TargetID   snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Target     *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
	Muting     bool         `gorm:"not null;default:false"`
	Blocking   bool         `gorm:"not null;default:false"`
	BlockedBy  bool         `gorm:"not null;default:false"`
	Following  bool         `gorm:"not null;default:false"`
	FollowedBy bool         `gorm:"not null;default:false"`
}

// BeforeUpdate creates a relationship between the actor and target if needed.
func (r *Relationship) BeforeUpdate(tx *gorm.DB) error {
	var original Relationship
	if err := tx.First(&original, "actor_id = ? and target_id = ?", r.ActorID, r.TargetID).Error; err != nil {
		return err
	}
	fmt.Printf("relationship changed from %+v to %+v", original, r)
	// what changed?
	switch {
	case original.Following && !r.Following:
		// unfollow
		return tx.Create(&RelationshipRequest{
			ActorID:  r.ActorID,
			TargetID: r.TargetID,
			Action:   "unfollow",
		}).Error
	case !original.Following && r.Following:
		// follow
		return tx.Create(&RelationshipRequest{
			ActorID:  r.ActorID,
			TargetID: r.TargetID,
			Action:   "follow",
		}).Error
	default:
		return nil
	}
}

// AfterUpdate updates the followers and following counts for the actor and target.
func (r *Relationship) AfterUpdate(tx *gorm.DB) error {
	return forEach(tx, r.updateFollowersCount, r.updateFollowingCount)
}

// updateFollowersCount updates the followers count for the target.
func (r *Relationship) updateFollowersCount(tx *gorm.DB) error {
	actor := &Actor{
		ID: snowflake.ID(r.ActorID),
	}
	followers := tx.Select("COUNT(*)").Where("target_id = ? and following = true", r.ActorID).Table("relationships")
	return tx.Model(actor).Update("followers_count", followers).Error
}

// updateFollowingCount updates the following count for the actor.
func (r *Relationship) updateFollowingCount(tx *gorm.DB) error {
	actor := &Actor{
		ID: snowflake.ID(r.TargetID),
	}
	following := tx.Select("COUNT(*)").Where("actor_id = ? and following = true", r.TargetID).Table("relationships")
	return tx.Model(actor).Update("following_count", following).Error
}

// A RelationshipRequest records a request to follow or unfollow an actor.
// RelationshipRequests are created by hooks on the Relationship model, and are
// processed by the RelationshipRequestProcessor in the background.
type RelationshipRequest struct {
	ID uint32 `gorm:"primarykey;"`
	// CreatedAt is the time the request was created.
	CreatedAt time.Time
	// UpdatedAt is the time the request was last updated.
	UpdatedAt time.Time
	ActorID   snowflake.ID `gorm:"uniqueIndex:idx_actor_id_target_id;not null;"`
	// Actor is the actor that is requesting the relationship change.
	Actor    *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
	TargetID snowflake.ID `gorm:"uniqueIndex:idx_actor_id_target_id;not null;"`
	// Target is the actor that is being followed or unfollowed.
	Target *Actor `gorm:"constraint:OnDelete:CASCADE;"`
	// Action is the action to perform, either follow or unfollow.
	Action string `gorm:"type:enum('follow', 'unfollow');not null"`
	// Attempts is the number of times the request has been attempted.
	Attempts uint32 `gorm:"not null;default:0"`
	// LastAttempt is the time the request was last attempted.
	LastAttempt time.Time
	// LastResult is the result of the last attempt if it failed.
	LastResult string `gorm:"size:255;serializer:json"`
}
