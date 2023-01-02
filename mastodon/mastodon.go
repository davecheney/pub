// Package mastodon implements a Mastodon API service.
package mastodon

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/davecheney/pub/internal/models"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

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

func (s *Service) Instances() *Instances {
	return &Instances{
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

func (s *Service) Relationships() *Relationships {
	return &Relationships{
		service: s,
	}
}

func (s *Service) Search() *Search {
	return &Search{
		service: s,
	}
}

func (s *Service) Statuses() *Statuses {
	return &Statuses{
		service: s,
	}
}

func (s *Service) Timelines() *Timelines {
	return &Timelines{
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

// toJSON writes the given object to the response body as JSON.
// If obj is a nil slice, an empty JSON array is written.
// If obj is a nil map, an empty JSON object is written.
// If obj is a nil pointer, a null is written.
func toJSON(w http.ResponseWriter, obj any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.MarshalOptions{}.MarshalFull(json.EncodeOptions{
		Indent: "  ",
	}, w, obj)
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
