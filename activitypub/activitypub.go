package activitypub

import (
	"crypto"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

// Service represents the Service REST resource.
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) Followers() *Followers     { return &Followers{service: s} }
func (s *Service) Following() *Following     { return &Following{service: s} }
func (s *Service) Collections() *Collections { return &Collections{service: s} }
func (s *Service) Inboxes(getKey func(keyId string) (crypto.PublicKey, error)) *Inboxes {
	return &Inboxes{
		service: s,
		getKey:  getKey,
	}
}
func (s *Service) Outboxes() *Outbox { return &Outbox{service: s} }

type Followers struct {
	service *Service
}

func (s *Service) Users() *Users {
	return &Users{
		service: s,
	}
}

func (f *Followers) Index(w http.ResponseWriter, r *http.Request) {
	toJSON(w, map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("https://%s/users/%s/followers", r.Host, chi.URLParam(r, "username")),
		"type":         "OrderedCollection",
		"totalItems":   0,
		"orderedItems": []any{},
	})
}

type Following struct {
	service *Service
}

func (f *Following) Index(w http.ResponseWriter, r *http.Request) {
	toJSON(w, map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("https://%s/users/%s/following", r.Host, chi.URLParam(r, "username")),
		"type":         "OrderedCollection",
		"totalItems":   0,
		"orderedItems": []any{},
	})
}

type Collections struct {
	service *Service
}

func (f *Collections) Show(w http.ResponseWriter, r *http.Request) {
	toJSON(w, map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("https://%s/users/%s/collections/%s", r.Host, chi.URLParam(r, "username"), chi.URLParam(r, "collection")),
		"type":         "OrderedCollection",
		"totalItems":   0,
		"orderedItems": []any{},
	})
}

type Outbox struct {
	service *Service
}

func (o *Outbox) Index(w http.ResponseWriter, r *http.Request) {
	toJSON(w, map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("https://%s/users/%s/outbox", r.Host, chi.URLParam(r, "username")),
		"type":         "OrderedCollection",
		"totalItems":   0,
		"orderedItems": []any{},
	})
}

// toJSON writes the given object to the response body as JSON.
func toJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return json.MarshalFull(w, obj)
}
