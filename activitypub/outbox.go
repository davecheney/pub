package activitypub

import (
	"fmt"
	"net/http"

	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
)

func OutboxIndex(w http.ResponseWriter, r *http.Request) {
	to.JSON(w, map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("https://%s/users/%s/outbox", r.Host, chi.URLParam(r, "username")),
		"type":         "OrderedCollection",
		"totalItems":   0,
		"orderedItems": []any{},
	})
}
