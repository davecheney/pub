package m

import (
	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

type conversations struct {
	db      *gorm.DB
	service *Service
}

// New returns a new conversations with the given visibility.
func (c *conversations) New(vis string) (*models.Conversation, error) {
	conv := models.Conversation{
		Visibility: vis,
	}
	if err := c.db.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (c *conversations) FindOrCreate(id uint32, vis string) (*models.Conversation, error) {
	var conversation models.Conversation
	if err := c.db.FirstOrCreate(&conversation, models.Conversation{
		Visibility: vis,
	}).Error; err != nil {
		return nil, err
	}
	return &conversation, nil
}
