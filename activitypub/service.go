package activitypub

import (
	"github.com/davecheney/m/m"
	"gorm.io/gorm"
)

// Service implements a Mastodon service.
type Service struct {
	db      *gorm.DB
	service *m.Service
}

// NewService returns a new instance of Service.
func NewService(db *gorm.DB, service *m.Service) *Service {
	return &Service{
		db:      db,
		service: service,
	}
}

func (svc *Service) accounts() *m.Accounts {
	return svc.service.API().Accounts() //
}
