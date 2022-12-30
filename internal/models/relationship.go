package models

import (
	"gorm.io/gorm"
)

type Relationship struct {
	ActorID    uint64 `gorm:"primarykey;autoIncrement:false"`
	TargetID   uint64 `gorm:"primarykey;autoIncrement:false"`
	Target     *Actor
	Muting     bool `gorm:"not null;default:false"`
	Blocking   bool `gorm:"not null;default:false"`
	BlockedBy  bool `gorm:"not null;default:false"`
	Following  bool `gorm:"not null;default:false"`
	FollowedBy bool `gorm:"not null;default:false"`
}

func (r *Relationship) AfterUpdate(tx *gorm.DB) error {
	return withTX(tx, r.updateFollowersCount, r.updateFollowingCount)
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
