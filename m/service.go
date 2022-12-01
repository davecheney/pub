package m

import "gorm.io/gorm"

// Service represents the m web service.
type Service struct {
	db *gorm.DB
	// The Instance this service represents.
	instance *Instance
}

// NewService returns a new Service.
func NewService(db *gorm.DB, domain string) (*Service, error) {
	var instance Instance
	if err := db.Where("domain = ?", domain).First(&instance).Error; err != nil {
		return nil, err
	}
	return &Service{
		db:       db,
		instance: &instance,
	}, nil
}

func (s *Service) API() *API {
	return &API{
		service: s,
	}
}

func (s *Service) Contexts() *Contexts {
	return &Contexts{
		service: s,
	}
}

// Domain returns the domain of the instance.
func (s *Service) Domain() string {
	return s.instance.Domain
}

// NodeInfo returns a NodeInfo REST resource.
func (s *Service) NodeInfo() *NodeInfo {
	return &NodeInfo{
		db:     s.db,
		domain: s.Domain(),
	}
}

func (s *Service) OAuth() *OAuth {
	return &OAuth{
		db:       s.db,
		instance: s.instance,
	}
}

func (s *Service) WellKnown() *WellKnown {
	return &WellKnown{
		db:       s.db,
		instance: s.instance,
	}
}

func (s *Service) tokens() *tokens {
	return &tokens{
		db: s.db,
	}
}

func (s *Service) Statuses() *statuses {
	return &statuses{
		db:      s.db,
		service: s,
	}
}

func (s *Service) conversations() *conversations {
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

func (s *Service) instances() *instances {
	return &instances{
		db: s.db,
	}
}

// API rerpesents the root of a Mastodon capable REST API.
type API struct {
	service *Service
}

func (a *API) Accounts() *Accounts {
	return &Accounts{
		db:      a.service.db,
		service: a.service,
	}
}

func (a *API) Applications() *Applications {
	return &Applications{
		db:       a.service.db,
		instance: a.service.instance,
	}
}

func (a *API) Conversations() *Conversations {
	return &Conversations{
		db:      a.service.db,
		service: a.service,
	}
}

func (a *API) Emojis() *Emojis {
	return &Emojis{
		db: a.service.db,
	}
}

func (a API) Filters() *Filters {
	return &Filters{
		db: a.service.db,
	}
}

func (a *API) Instances() *Instances {
	return &Instances{
		db:       a.service.db,
		instance: a.service.instance,
	}
}

func (a *API) Lists() *Lists {
	return &Lists{
		db:       a.service.db,
		instance: a.service.instance,
	}
}

func (a *API) Notifications() *Notifications {
	return &Notifications{
		db: a.service.db,
	}
}

func (a *API) Statuses() *Statuses {
	return &Statuses{
		db:      a.service.db,
		service: a.service,
	}
}

func (a *API) Timelines() *Timelines {
	return &Timelines{
		db:       a.service.db,
		instance: a.service.instance,
	}
}
