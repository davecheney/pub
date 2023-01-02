package activitypub

import (
	"crypto"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/to"
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

func (s *Service) Collections() *Collections { return &Collections{service: s} }
func (s *Service) Inboxes(getKey func(keyId string) (crypto.PublicKey, error)) *Inboxes {
	return &Inboxes{
		service: s,
		getKey:  getKey,
	}
}

func FollowersIndex(w http.ResponseWriter, r *http.Request) {
	to.JSON(w, map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("https://%s/users/%s/followers", r.Host, chi.URLParam(r, "username")),
		"type":         "OrderedCollection",
		"totalItems":   0,
		"orderedItems": []any{},
	})
}

func FollowingIndex(w http.ResponseWriter, r *http.Request) {
	to.JSON(w, map[string]any{
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
	to.JSON(w, map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("https://%s/users/%s/collections/%s", r.Host, chi.URLParam(r, "username"), chi.URLParam(r, "collection")),
		"type":         "OrderedCollection",
		"totalItems":   0,
		"orderedItems": []any{},
	})
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func timeFromAnyOrZero(v any) time.Time {
	switch v := v.(type) {
	case string:
		t, _ := time.Parse(time.RFC3339, v)
		return t
	case time.Time:
		return v
	default:
		return time.Time{}
	}
}

func timeFromAny(v any) (time.Time, error) {
	switch v := v.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	case time.Time:
		return v, nil
	default:
		return time.Time{}, errors.New("timeFromAny: invalid type")
	}
}

func intFromAny(v any) int {
	switch v := v.(type) {
	case int:
		return v
	case float64:
		// shakes fist at json number type
		return int(v)
	}
	return 0
}

func anyToSlice(v any) []any {
	switch v := v.(type) {
	case []any:
		return v
	default:
		return nil
	}
}

func marshalIndent(v any) ([]byte, error) {
	b, err := json.MarshalOptions{}.Marshal(json.EncodeOptions{
		Indent: "\t", // indent for readability
	}, v)
	return b, err
}
