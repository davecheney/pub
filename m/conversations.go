package m

import (
	"time"

	"gorm.io/gorm"
)

type Conversation struct {
	ID         uint `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Visibility string `gorm:"type:enum('public', 'unlisted', 'private', 'direct', 'limited');not null"`
	Statuses   []Status
}

type conversations struct {
	db      *gorm.DB
	service *Service
}

func (c *conversations) FindConversationByStatusID(id uint64) (*Conversation, error) {
	var status Status
	if err := c.db.Where("id = ?", id).First(&status).Error; err != nil {
		return nil, err
	}
	var conversation Conversation
	if err := c.db.Preload("Statuses").First(&conversation, status.ConversationID).Error; err != nil {
		return nil, err
	}
	return &conversation, nil
}
