package m

import (
	"github.com/davecheney/m/internal/models"
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

func (s *Service) Instances() *instances {
	return &instances{
		db: s.db,
	}
}

func (s *Service) Reactions() *reactions {
	return &reactions{
		db: s.db,
	}
}

type reactions struct {
	db *gorm.DB
}

func (r *reactions) Pin(status *models.Status, actor *models.Actor) error {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return err
	}
	reaction.Pinned = true
	return r.db.Model(reaction).Update("pinned", true).Error
}

func (r *reactions) Unpin(status *models.Status, actor *models.Actor) error {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return err
	}
	reaction.Pinned = false
	return r.db.Model(reaction).Update("pinned", false).Error
}

func (r *reactions) Favourite(status *models.Status, actor *models.Actor) (*models.Reaction, error) {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Favourited = true
	if err := r.db.Model(reaction).Update("favourited", true).Error; err != nil {
		return nil, err
	}
	return reaction, nil
}

func (r *reactions) Unfavourite(status *models.Status, actor *models.Actor) (*models.Reaction, error) {
	reaction, err := r.findOrCreate(status, actor)
	if err != nil {
		return nil, err
	}
	reaction.Favourited = false
	if err := r.db.Model(reaction).Update("favourited", false).Error; err != nil {
		return nil, err
	}
	return reaction, nil
}

func (r *reactions) findOrCreate(status *models.Status, actor *models.Actor) (*models.Reaction, error) {
	var reaction models.Reaction
	if err := r.db.FirstOrCreate(&reaction, models.Reaction{StatusID: status.ID, ActorID: actor.ID}).Error; err != nil {
		return nil, err
	}
	return &reaction, nil
}

func (s *Service) Relationships() *relationships {
	return &relationships{
		db: s.db,
	}
}

type relationships struct {
	db *gorm.DB
}

// Block blocks the target from the actor.
func (r *relationships) Block(actor, target *models.Actor) (*models.Relationship, error) {
	forward, inverse, err := r.pair(actor, target)
	if err != nil {
		return nil, err
	}
	forward.Blocking = true
	if err := r.db.Model(forward).Update("blocking", true).Error; err != nil {
		return nil, err
	}
	inverse.BlockedBy = true
	if err := r.db.Model(inverse).Update("blocked_by", true).Error; err != nil {
		return nil, err
	}
	return forward, nil
}

// Unblock removes a block relationship between actor and the target.
func (r *relationships) Unblock(actor, target *models.Actor) (*models.Relationship, error) {
	forward, inverse, err := r.pair(actor, target)
	if err != nil {
		return nil, err
	}
	forward.Blocking = false
	if err := r.db.Model(forward).Update("blocking", false).Error; err != nil {
		return nil, err
	}
	inverse.BlockedBy = false
	if err := r.db.Model(inverse).Update("blocked_by", false).Error; err != nil {
		return nil, err
	}
	return forward, nil
}

// Follow establishes a follow relationship between actor and the target.
func (r *relationships) Follow(actor, target *models.Actor) (*models.Relationship, error) {
	forward, inverse, err := r.pair(actor, target)
	if err != nil {
		return nil, err
	}
	// this magic is important, updating the local copy, then passing it to db.Model makes it
	// available to the BeforeCreate hook. Then the hook can check how the relationship has changed
	// compared to the previous state.
	forward.Following = true
	if err := r.db.Model(forward).Update("following", true).Error; err != nil {
		return nil, err
	}
	inverse.FollowedBy = true
	if err := r.db.Model(inverse).Update("followed_by", true).Error; err != nil {
		return nil, err
	}
	return forward, nil
}

// Unfollow removes a follow relationship between actor and the target.
func (r *relationships) Unfollow(actor, target *models.Actor) (*models.Relationship, error) {
	forward, inverse, err := r.pair(actor, target)
	if err != nil {
		return nil, err
	}
	forward.Following = false
	if err := r.db.Model(forward).Update("following", false).Error; err != nil {
		return nil, err
	}
	inverse.FollowedBy = false
	if err := r.db.Model(inverse).Update("followed_by", false).Error; err != nil {
		return nil, err
	}
	return forward, nil
}

// pair returns the pair of relationships between actor and target.
func (r *relationships) pair(actor, target *models.Actor) (*models.Relationship, *models.Relationship, error) {
	forward, err := r.findOrCreate(actor, target)
	if err != nil {
		return nil, nil, err
	}
	inverse, err := r.findOrCreate(target, actor)
	if err != nil {
		return nil, nil, err
	}
	return forward, inverse, nil
}

func (r *relationships) findOrCreate(actor, target *models.Actor) (*models.Relationship, error) {
	var rel models.Relationship
	if err := r.db.FirstOrCreate(&rel, models.Relationship{ActorID: actor.ID, TargetID: target.ID}).Error; err != nil {
		return nil, err
	}
	rel.Target = target
	return &rel, nil
}
