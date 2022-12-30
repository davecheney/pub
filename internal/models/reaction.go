package models

import (
	"gorm.io/gorm"
)

// Reaction represents an an actors reaction to a status.
type Reaction struct {
	StatusID   uint64 `gorm:"primarykey;autoIncrement:false"`
	Status     *Status
	ActorID    uint64 `gorm:"primarykey;autoIncrement:false"`
	Actor      *Actor
	Favourited bool `gorm:"not null;default:false"`
	Reblogged  bool `gorm:"not null;default:false"`
	Muted      bool `gorm:"not null;default:false"`
	Bookmarked bool `gorm:"not null;default:false"`
	Pinned     bool `gorm:"not null;default:false"`
}

func (r *Reaction) AfterUpdate(tx *gorm.DB) error {
	return withTX(tx, r.updateStatusCount)
}

// updateStatusCount updates the favourites_count and reblogs_count fields on the status.
func (r *Reaction) updateStatusCount(tx *gorm.DB) error {
	status := &Status{ID: r.StatusID}
	favouritesCount := tx.Select("COUNT(*)").Where("status_id = ? and favourited = true", r.StatusID).Table("reactions")
	reblogsCount := tx.Select("COUNT(*)").Where("status_id = ? and reblogged = true", r.StatusID).Table("reactions")
	return tx.Model(status).Updates(map[string]interface{}{
		"favourites_count": favouritesCount,
		"reblogs_count":    reblogsCount,
	}).Error
}
