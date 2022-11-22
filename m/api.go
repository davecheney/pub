package m

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/davecheney/m/mastodon"
	"github.com/go-fed/httpsig"
	"github.com/gorilla/mux"
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

func (a *Api) StoreActivity(activity map[string]interface{}) error {
	b, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	object, _ := activity["object"].(map[string]interface{})
	objectType, _ := object["type"].(string)
	if _, err := tx.Exec("INSERT INTO activitypub_inbox (activity_type, object_type, activity) VALUES (?,?,?)", activity["type"], objectType, b); err != nil {
		log.Println("storeActivity:", err)
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func query[K comparable, V any](m map[K]V, k K) V {
	v, _ := m[k]
	return v
}

func (a *Api) ValidateSignature() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verifier, err := httpsig.NewVerifier(r)
			if err != nil {
				log.Println("validateSignature:", err)
			}
			log.Println("keyId:", verifier.KeyId())
			pubKey, err := a.GetKey(verifier.KeyId())
			if err != nil {
				log.Println("getkey:", err)
			}
			if err := verifier.Verify(pubKey, httpsig.RSA_SHA256); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a *Api) GetKey(id string) (crypto.PublicKey, error) {
	actor_id := trimKeyId(id)
	if actor, err := a.findActorById(actor_id); err == nil {
		return pemToPublicKey(actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string))
	} else {
		log.Println("findActorById:", err)
	}

	actor, err := fetchActor(actor_id)
	if err != nil {
		return nil, err
	}
	if err := a.saveActor(actor); err != nil {
		return nil, err
	}
	return pemToPublicKey(actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string))
}

// fetchActor fetches an actor from the remote server.
func fetchActor(id string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", id, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: newrequest: %w", err)
	}
	req.Header.Set("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: do: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchActor: status: %d", resp.StatusCode)
	}
	var v map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("fetchActor: jsondecode: %w", err)
	}
	return v, nil
}

// pemToPublicKey converts a PEM encoded public key to a crypto.PublicKey.
func pemToPublicKey(pemEncoded string) (crypto.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
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

func (a *Api) findActorById(id string) (map[string]any, error) {
	var b []byte
	err := a.db.QueryRowx("SELECT object FROM activitypub_actors WHERE actor_id = ? ORDER BY created_at desc LIMIT 1", id).Scan(&b)
	if err != nil {
		return nil, err
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (a *Api) saveActor(actor map[string]interface{}) error {
	b, err := json.Marshal(actor)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("INSERT INTO activitypub_actors (actor_id, type, object, publickey) VALUES (?,?,?,?)", actor["id"], actor["type"], b, actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string)); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *Api) UsersShow(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	actor_id := fmt.Sprintf("https://cheney.net/users/%s", username)
	actor, err := a.findActorById(actor_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(actor)
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
