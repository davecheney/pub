// Package mastodon implements a Mastodon API service.
package mastodon

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/davecheney/m/m"
	"github.com/go-json-experiment/json"
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

func (s *Service) Contexts() *Contexts {
	return &Contexts{
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

// toJSON writes the given object to the response body as JSON.
func toJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return json.MarshalFull(w, obj)
}

func stringOrDefault(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

type number interface {
	uint | uint64
}

func stringOrNull[T number](v *T) any {
	if v == nil {
		return nil
	}
	return strconv.Itoa(int(*v))
}

func toString[T number](n T) string {
	return strconv.FormatInt(int64(n), 10)
}
