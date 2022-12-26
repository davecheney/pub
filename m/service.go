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

func (s *Service) Actors() *Actors {
	return &Actors{
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

func (s *Service) Relationships() *relationships {
	return &relationships{
		db: s.db,
	}
}

type relationships struct {
	db *gorm.DB
}

// Follow establishes a follow relationship between actor and the target.
func (r *relationships) Follow(actor, target *Actor) (*Relationship, error) {
	forward, err := r.findOrCreate(actor, target)
	if err != nil {
		return nil, err
	}
	if err := r.db.Model(forward).Update("following", true).Error; err != nil {
		return nil, err
	}
	inverse, err := r.findOrCreate(target, actor)
	if err != nil {
		return nil, err
	}
	if err := r.db.Model(inverse).Update("followed_by", true).Error; err != nil {
		return nil, err
	}
	forward.Following = true
	return forward, nil
}

// Unfollow removes a follow relationship between actor and the target.
func (r *relationships) Unfollow(actor, target *Actor) (*Relationship, error) {
	forward, err := r.findOrCreate(actor, target)
	if err != nil {
		return nil, err
	}
	if err := r.db.Model(forward).Update("following", false).Error; err != nil {
		return nil, err
	}
	inverse, err := r.findOrCreate(target, actor)
	if err != nil {
		return nil, err
	}
	if err := r.db.Model(inverse).Update("followed_by", false).Error; err != nil {
		return nil, err
	}
	forward.Following = false
	return forward, nil
}

func (r *relationships) findOrCreate(actor, target *Actor) (*Relationship, error) {
	var rel Relationship
	err := r.db.Joins("Target").First(&rel, "actor_id = ? and target_id = ?", actor.ID, target.ID).Error
	if err == nil {
		return &rel, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	rel = Relationship{
		ActorID:  actor.ID,
		TargetID: target.ID,
		Target:   target,
	}
	result := r.db.Create(&rel)
	return &rel, result.Error
}
