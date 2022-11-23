package activitypub

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-fed/httpsig"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

// Service implements a Mastodon service.
type Service struct {
	db *sqlx.DB
}

// NewService returns a new instance of Service.
func NewService(db *sqlx.DB) *Service {
	return &Service{
		db: db,
	}
}

func (svc *Service) actors() *actors {
	return &actors{db: svc.db}
}

func (svc *Service) activities() *activities {
	return &activities{db: svc.db}
}

func (svc *Service) InboxCreate(w http.ResponseWriter, r *http.Request) {
	var activity map[string]any
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := svc.activities().create(activity); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (svc *Service) UsersShow(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	actor_id := fmt.Sprintf("https://cheney.net/users/%s", username)
	actor, err := svc.actors().findById(actor_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(actor)
}

func (svc *Service) ValidateSignature() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verifier, err := httpsig.NewVerifier(r)
			if err != nil {
				log.Println("validateSignature:", err)
			}
			log.Println("keyId:", verifier.KeyId())
			pubKey, err := svc.getKey(verifier.KeyId())
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

func (svc *Service) getKey(id string) (crypto.PublicKey, error) {
	actor_id := trimKeyId(id)
	if actor, err := svc.actors().findById(actor_id); err == nil {
		return pemToPublicKey(actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string))
	} else {
		log.Println("findActorById:", err)
	}

	actor, err := fetchActor(actor_id)
	if err != nil {
		return nil, err
	}
	if err := svc.actors().create(actor); err != nil {
		return nil, err
	}
	return pemToPublicKey(actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string))
}

// trimKeyId removes the #main-key suffix from the key id.
func trimKeyId(id string) string {
	if i := strings.Index(id, "#"); i != -1 {
		return id[:i]
	}
	return id
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
