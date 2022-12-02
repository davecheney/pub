package m

import (
	"fmt"
	"net/http"

	"github.com/go-json-experiment/json"
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

type Followers struct {
	service *Service
}

func (f *Followers) Index(w http.ResponseWriter, r *http.Request) {
	user, err := f.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	following := f.service.db.Model(&user).Association("Following").Count()

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, map[string]any{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"id":         fmt.Sprintf("https://%s/users/%s/followers", r.Host, user.Username),
		"type":       "OrderedCollection",
		"totalItems": following,
		"first":      fmt.Sprintf("https://%s/users/%s/followers?page=1", r.Host, user.Username),
	})
}

type Following struct {
	service *Service
}

func (f *Following) Index(w http.ResponseWriter, r *http.Request) {
	_, err := f.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[]`))
}
