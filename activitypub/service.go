package activitypub

import (
	"crypto"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-json-experiment/json"

	"github.com/go-fed/httpsig"
	"gorm.io/gorm"
)

// Service implements a Mastodon service.
type Service struct {
	db *gorm.DB
}

// NewService returns a new instance of Service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (svc *Service) ValidateSignature() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verifier, err := httpsig.NewVerifier(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			pubKey, err := svc.GetKey(verifier.KeyId())
			if err != nil {
				log.Println("getkey:", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := verifier.Verify(pubKey, httpsig.RSA_SHA256); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (svc *Service) GetKey(id string) (crypto.PublicKey, error) {
	actor_id := trimKeyId(id)
	var actor Actor
	err := svc.db.Where("actor_id = ?", actor_id).First(&actor).Error
	if err == nil {
		// found cached key
		return actor.pemToPublicKey()
	}
	actor_, err := fetchActor(actor_id)
	if err != nil {
		return nil, err
	}
	object, err := json.Marshal(actor_)
	if err != nil {
		return nil, fmt.Errorf("getKey: marshal: %w", err)
	}
	actor = Actor{
		ActorID:   actor_["id"].(string),
		Type:      actor_["type"].(string),
		Object:    object,
		PublicKey: actor_["publicKey"].(map[string]interface{})["publicKeyPem"].(string),
	}
	if err := svc.db.Create(&actor).Error; err != nil {
		return nil, fmt.Errorf("getKey: create: %w", err)
	}

	return actor.pemToPublicKey()
}

// trimKeyId removes the #main-key suffix from the key id.
func trimKeyId(id string) string {
	if i := strings.Index(id, "#"); i != -1 {
		return id[:i]
	}
	return id
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
		return nil, fmt.Errorf("fetchActor: %v do: %w", req.URL, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchActor: %v status: %d", resp.Request.URL, resp.StatusCode)
	}
	var v map[string]interface{}
	if err := json.UnmarshalFull(resp.Body, &v); err != nil {
		return nil, fmt.Errorf("fetchActor: jsondecode: %w", err)
	}
	return v, nil
}
