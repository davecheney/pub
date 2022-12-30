package models

import (
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

func (r *Relationship) AfterUpdate(tx *gorm.DB) error {
	return withTX(tx, r.updateFollowersCount, r.updateFollowingCount)
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
