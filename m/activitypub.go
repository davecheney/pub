package m

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ActivityPub represents the ActivityPub REST resource.
type ActivityPub struct {
	service *Service
}

func (a *ActivityPub) Followers() *Followers {
	return &Followers{service: a.service}
}

func (a *ActivityPub) Following() *Following {
	return &Following{service: a.service}
}

func (a *ActivityPub) Collections() *Collections {
	return &Collections{service: a.service}
}

type Followers struct {
	service *Service
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
