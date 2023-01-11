package activitypub

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
)

type Env struct {
	*models.Env
}

func (e *Env) GetKey(keyID string) (crypto.PublicKey, error) {

	// defer resolving the admin actor until we need use it to fetch the remote actor
	fetch := func(uri string) (*models.Actor, error) {
		var instance models.Instance
		if err := e.DB.Joins("Admin").Preload("Admin.Actor").Take(&instance, "admin_id is not null").Error; err != nil {
			return nil, err
		}
		fetcher := NewRemoteActorFetcher(instance.Admin, e.DB)
		return fetcher.Fetch(uri)
	}

	actor, err := models.NewActors(e.DB).FindOrCreate(trimKeyId(keyID), fetch)
	if err != nil {
		return nil, err
	}
	return pemToPublicKey(actor.PublicKey)
}

func pemToPublicKey(key []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("pemToPublicKey: invalid pem type: %s", block.Type)
	}
	var publicKey interface{}
	var err error
	if publicKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("pemToPublicKey: parsepkixpublickey: %w", err)
	}
	return publicKey, nil
}

// trimKeyId removes the #main-key suffix from the key id.
func trimKeyId(id string) string {
	if i := strings.Index(id, "#"); i != -1 {
		return id[:i]
	}
	return id
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

func CollectionsShow(w http.ResponseWriter, r *http.Request) {
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
