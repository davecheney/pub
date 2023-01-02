// Package mastodon implements a Mastodon API service.
package mastodon

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"gorm.io/gorm"
)

type Env struct {
	// DB is the database connection.
	DB *gorm.DB
}

// authenticate authenticates the bearer token attached to the request and, if
// successful, returns the account associated with the token.
func (e *Env) authenticate(r *http.Request) (*models.Account, error) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token models.Token
	if err := e.DB.Joins("Account").Preload("Account.Actor").Preload("Account.Role").First(&token, "access_token = ?", bearer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Error(http.StatusUnauthorized, err)
		}
		return nil, err
	}
	return token.Account, nil
}

func (e *Env) findByDomain(domain string) (*models.Instance, error) {
	var instance models.Instance
	if err := e.DB.Where("domain = ?", domain).Preload("Admin").Preload("Admin.Actor").Preload("Rules").First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Error(http.StatusNotFound, err)
		}
		return nil, err
	}
	return &instance, nil
}

// Service represents a Mastodon API service.
type Service struct {
	db *gorm.DB
}

// NewService returns a new Mastodon API service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) Accounts() *Accounts {
	return &Accounts{
		service: s,
	}
}

func (s *Service) Applications() *Applications {
	return &Applications{
		service: s,
	}
}

func (s *Service) Blocks() *Blocks {
	return &Blocks{
		service: s,
	}
}

func (s *Service) Contexts() *Contexts {
	return &Contexts{
		service: s,
	}
}

func (s *Service) Conversations() *Conversations {
	return &Conversations{
		service: s,
	}
}

func (s *Service) Directory() *Directory {
	return &Directory{
		service: s,
	}
}

func (s *Service) Emojis() *Emojis {
	return &Emojis{
		service: s,
	}
}

func (s *Service) Favourites() *Favourites {
	return &Favourites{
		service: s,
	}
}

func (s *Service) Filters() *Filters {
	return &Filters{
		service: s,
	}
}

func (s *Service) Lists() *Lists {
	return &Lists{
		service: s,
	}
}

func (s *Service) Markers() *Markers {
	return &Markers{
		service: s,
	}
}

func (s *Service) Mutes() *Mutes {
	return &Mutes{
		service: s,
	}
}

func (s *Service) Notifications() *Notifications {
	return &Notifications{
		service: s,
	}
}

func (s *Service) Statuses() *Statuses {
	return &Statuses{
		service: s,
	}
}

// authenticate authenticates the bearer token attached to the request and, if
// successful, returns the account associated with the token.
func (s *Service) authenticate(r *http.Request) (*models.Account, error) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token models.Token
	if err := s.db.Joins("Account").Preload("Account.Actor").Preload("Account.Role").First(&token, "access_token = ?", bearer).Error; err != nil {
		return nil, err
	}
	return token.Account, nil
}

func stringOrDefault(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

func stringOrNull[T number](v *T) any {
	if v == nil {
		return nil
	}
	return strconv.Itoa(int(*v))
}

type number interface {
	~uint | ~uint64 | ~uint32
}

func toString[T number](n T) string {
	return strconv.FormatUint(uint64(n), 10)
}

func utoa(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}
