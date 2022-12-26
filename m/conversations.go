package m

import (
	"gorm.io/gorm"
)

type conversations struct {
	db      *gorm.DB
	service *Service
}

// New returns a new conversations with the given visibility.
func (c *conversations) New(vis string) (*Conversation, error) {
	conv := Conversation{
		Visibility: vis,
	}
	if err := c.db.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (c *conversations) FindOrCreate(id uint32, vis string) (*Conversation, error) {
	var conversation Conversation
	if err := c.db.FirstOrCreate(&conversation, Conversation{Visibility: vis}).Error; err != nil {
		return nil, err
	}
	return &conversation, nil
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
