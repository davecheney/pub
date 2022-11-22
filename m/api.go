package m

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/davecheney/m/mastodon"
	"github.com/jmoiron/sqlx"

	_ "embed"
)

//go:embed public.pem
var publicKey string

func New(db *sqlx.DB) *Api {
	return &Api{
		db: db,
	}
}

type Api struct {
	db *sqlx.DB
}

func query[K comparable, V any](m map[K]V, k K) V {
	v, _ := m[k]
	return v
}

func (a *Api) InstanceFetch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&mastodon.Instance{
		URI:              "https://cheney.net/",
		Title:            "Casa del Cheese",
		ShortDescription: "🧀",
		Email:            "dave@cheney.net",
		Version:          "0.1.2",
		Languages:        []string{"en"},
	})
}

func (a *Api) InstancePeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]string{})
}

func (a *Api) TimelinesHome(w http.ResponseWriter, r *http.Request) {
	since, _ := strconv.ParseInt(r.FormValue("since_id"), 10, 64)
	limit, _ := strconv.ParseInt(r.FormValue("limit"), 10, 64)
	rows, err := a.db.Queryx("SELECT id, activity FROM activitypub_inbox WHERE activity_type=? AND object_type=? AND id > ? ORDER BY created_at DESC LIMIT ?", "Create", "Note", since, limit)

	var statuses []mastodon.Status
	for rows.Next() {
		var entry string
		var id int
		if err = rows.Scan(&id, &entry); err != nil {
			break
		}
		var activity map[string]any
		json.NewDecoder(strings.NewReader(entry)).Decode(&activity)
		object, _ := activity["object"].(map[string]interface{})
		statuses = append(statuses, mastodon.Status{
			Id:         strconv.Itoa(id),
			Uri:        object["atomUri"].(string),
			CreatedAt:  object["published"].(string),
			Content:    object["content"].(string),
			Visibility: "public",
		})
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(statuses) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func (a *Api) WellknownWebfinger(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	if resource != "acct:dave@cheney.net" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/jrd+json")
	json.NewEncoder(w).Encode(map[string]any{
		"subject": resource,
		"links": []map[string]any{
			{
				"rel":  "http://webfinger.net/rel/profile-page",
				"type": "text/html",
				"href": "https://cheney.net/dave",
			},
			{
				"rel":  "self",
				"type": "application/activity+json",
				"href": "https://cheney.net/users/dave",
			},
			{
				"rel":      "http://ostatus.org/schema/1.0/subscribe",
				"template": "https://cheney.net/authorize_interaction?uri={uri}",
			},
		},
	})
}
