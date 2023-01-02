package models

import (
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// BeforeUpdate creates a relationship request between the actor and target.
func (r *Relationship) BeforeUpdate(tx *gorm.DB) error {
	var original Relationship
	if err := tx.First(&original, "actor_id = ? and target_id = ?", r.ActorID, r.TargetID).Error; err != nil {
		return err
	}
	fmt.Printf("relationship changed from %+v to %+v\n", original, r)

	// if there is a conflict; eg. a follow then an unfollow before the follow is processed
	// update the existing row to reflect the new action.
	tx = tx.Clauses(clause.OnConflict{
		UpdateAll: true,
	})

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
		ID: r.ActorID,
	}
	followers := tx.Select("COUNT(*)").Where("target_id = ? and following = true", r.ActorID).Table("relationships")
	return tx.Model(actor).Update("followers_count", followers).Error
}

// updateFollowingCount updates the following count for the actor.
func (r *Relationship) updateFollowingCount(tx *gorm.DB) error {
	actor := &Actor{
		ID: r.TargetID,
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
	LastResult string `gorm:"size:255;not null;default:''"`
}

type Relationships struct {
	db *gorm.DB
}

func NewRelationships(db *gorm.DB) *Relationships {
	return &Relationships{
		db: db,
	}
}

// Mute mutes the target from the actor.
func (r *Relationships) Mute(actor, target *Actor) (*Relationship, error) {
	forward, err := r.findOrCreate(actor, target)
	if err != nil {
		return nil, err
	}
	forward.Muting = true
	if err := r.db.Model(forward).Update("muting", true).Error; err != nil {
		return nil, err
	}
	// there is no inverse relationship for muting
	return forward, nil
}

// Unmute removes a mute relationship between actor and the target.
func (r *Relationships) Unmute(actor, target *Actor) (*Relationship, error) {
	forward, err := r.findOrCreate(actor, target)
	if err != nil {
		return nil, err
	}
	forward.Muting = false
	if err := r.db.Model(forward).Update("muting", false).Error; err != nil {
		return nil, err
	}
	// there is no inverse relationship for muting
	return forward, nil
}

// Block blocks the target from the actor.
func (r *Relationships) Block(actor, target *Actor) (*Relationship, error) {
	forward, inverse, err := r.pair(actor, target)
	if err != nil {
		return nil, err
	}
	forward.Blocking = true
	if err := r.db.Model(forward).Update("blocking", true).Error; err != nil {
		return nil, err
	}
	inverse.BlockedBy = true
	if err := r.db.Model(inverse).Update("blocked_by", true).Error; err != nil {
		return nil, err
	}
	return forward, nil
}

// Unblock removes a block relationship between actor and the target.
func (r *Relationships) Unblock(actor, target *Actor) (*Relationship, error) {
	forward, inverse, err := r.pair(actor, target)
	if err != nil {
		return nil, err
	}
	forward.Blocking = false
	if err := r.db.Model(forward).Update("blocking", false).Error; err != nil {
		return nil, err
	}
	inverse.BlockedBy = false
	if err := r.db.Model(inverse).Update("blocked_by", false).Error; err != nil {
		return nil, err
	}
	return forward, nil
}

// Follow establishes a follow relationship between actor and the target.
func (r *Relationships) Follow(actor, target *Actor) (*Relationship, error) {
	forward, inverse, err := r.pair(actor, target)
	if err != nil {
		return nil, err
	}
	// this magic is important, updating the local copy, then passing it to db.Model makes it
	// available to the BeforeCreate hook. Then the hook can check how the relationship has changed
	// compared to the previous state.
	forward.Following = true
	if err := r.db.Model(forward).Update("following", true).Error; err != nil {
		return nil, err
	}
	inverse.FollowedBy = true
	if err := r.db.Model(inverse).Update("followed_by", true).Error; err != nil {
		return nil, err
	}
	return forward, nil
}

// Unfollow removes a follow relationship between actor and the target.
func (r *Relationships) Unfollow(actor, target *Actor) (*Relationship, error) {
	forward, inverse, err := r.pair(actor, target)
	if err != nil {
		return nil, err
	}
	forward.Following = false
	if err := r.db.Model(forward).Update("following", false).Error; err != nil {
		return nil, err
	}
	inverse.FollowedBy = false
	if err := r.db.Model(inverse).Update("followed_by", false).Error; err != nil {
		return nil, err
	}
	return forward, nil
}

// pair returns the pair of Relationships between actor and target.
func (r *Relationships) pair(actor, target *Actor) (*Relationship, *Relationship, error) {
	forward, err := r.findOrCreate(actor, target)
	if err != nil {
		return nil, nil, err
	}
	inverse, err := r.findOrCreate(target, actor)
	if err != nil {
		return nil, nil, err
	}
	return forward, inverse, nil
}

func (r *Relationships) findOrCreate(actor, target *Actor) (*Relationship, error) {
	var rel Relationship
	if err := r.db.FirstOrCreate(&rel, Relationship{ActorID: actor.ID, TargetID: target.ID}).Error; err != nil {
		return nil, err
	}
	rel.Target = target
	return &rel, nil
}
