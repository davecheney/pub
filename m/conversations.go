package m

import (
	"gorm.io/gorm"
)

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
