// Package mastodon implements a Mastodon API service.
package mastodon

import (
	"net/http"
	"strings"

	"github.com/davecheney/m/m"
)

// Service represents a Mastodon API service.
type Service struct {
	*m.Service
}

// NewService returns a new Mastodon API service.
func NewService(s *m.Service) *Service {
	return &Service{
		Service: s,
	}
}

// authenticate authenticates the bearer token attached to the request and, if
// successful, returns the account associated with the token.
func (s *Service) authenticate(r *http.Request) (*m.Account, error) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token m.Token
	if err := s.DB().Where("access_token = ?", bearer).Joins("Account").First(&token).Error; err != nil {
		return nil, err
	}
	return token.Account, nil
}

func (s *Service) Markers() *Markers {
	return &Markers{
		service: s,
	}
}
