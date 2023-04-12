package activitypub

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/streaming"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Env struct {
	*gorm.DB
	*streaming.Mux
}

func Followers(env *Env, w http.ResponseWriter, r *http.Request) error {
	var followers []*models.Relationship
	query := env.DB.Joins("JOIN actors ON actors.id = relationships.target_id and actors.name = ? and actors.domain = ?", chi.URLParam(r, "name"), r.Host)
	if err := query.Model(&models.Relationship{}).Preload("Actor").Find(&followers, "following = true").Error; err != nil {
		return err
	}
	return to.JSON(w, map[string]any{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"id":         fmt.Sprintf("https://%s%s", r.Host, r.URL.Path),
		"type":       "OrderedCollection",
		"totalItems": len(followers),
		"orderedItems": algorithms.Map(
			followers,
			func(r *models.Relationship) string {
				return r.Actor.URI
			},
		),
	})
}

func Following(env *Env, w http.ResponseWriter, r *http.Request) error {
	var following []*models.Relationship
	query := env.DB.Joins("JOIN actors ON actors.id = relationships.actor_id and actors.name = ? and actors.domain = ?", chi.URLParam(r, "name"), r.Host)
	if err := query.Model(&models.Relationship{}).Preload("Target").Find(&following, "following = true").Error; err != nil {
		return err
	}
	return to.JSON(w, map[string]any{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"id":         fmt.Sprintf("https://%s%s", r.Host, r.URL.Path),
		"type":       "OrderedCollection",
		"totalItems": len(following),
		"orderedItems": algorithms.Map(
			following,
			func(r *models.Relationship) string {
				return r.Target.URI
			},
		),
	})
}

func CollectionsShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	var actor models.Actor
	if err := env.DB.Take(&actor, "name = ? and domain = ?", chi.URLParam(r, "name"), r.Host).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}

	return to.JSON(w, map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("https://%s%s", r.Host, r.URL.Path),
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

// parseBool parses a boolean value from a request parameter.
// If the parameter is not present, it returns false.
// If the parameter is present but cannot be parsed, it returns false
func parseBool(r *http.Request, key string) bool {
	switch r.URL.Query().Get(key) {
	case "true", "1":
		return true
	default:
		return false
	}
}
