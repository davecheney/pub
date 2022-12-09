package m

import (
	"gorm.io/gorm"
)

// Service represents the m web service.
type Service struct {
	db *gorm.DB
}

// NewService returns a new Service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) DB() *gorm.DB {
	return s.db
}

// NodeInfo returns a NodeInfo REST resource.
func (s *Service) NodeInfo() *NodeInfo {
	return &NodeInfo{
		db: s.db,
	}
}

func (s *Service) Users() *Users {
	return &Users{
		db:      s.db,
		service: s,
	}
}

func (s *Service) WellKnown() *WellKnown {
	return &WellKnown{
		db: s.db,
	}
}

func (s *Service) Statuses() *statuses {
	return &statuses{
		db:      s.db,
		service: s,
	}
}

func (s *Service) Conversations() *conversations {
	return &conversations{
		db:      s.db,
		service: s,
	}
}

func (s *Service) Accounts() *accounts {
	return &accounts{
		db:      s.db,
		service: s,
	}
}

func (s *Service) Instances() *instances {
	return &instances{
		db: s.db,
	}
}

func (s *Service) ActivityPub() *ActivityPub {
	return &ActivityPub{
		service: s,
	}
}

func (a *ActivityPub) Inboxes() *Inboxes {
	return &Inboxes{
		service: a.service,
	}
}

func (a *ActivityPub) Outboxes() *Outbox {
	return &Outbox{
		service: a.service,
	}
}
