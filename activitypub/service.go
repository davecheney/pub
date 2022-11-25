package activitypub

import (
	"crypto"
	"fmt"
	"log"
	"net/http"
	"strings"

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

func (svc *Service) actors() *Actors {
	return NewActors(svc.db)
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

func (svc *Service) GetKey(keyId string) (crypto.PublicKey, error) {
	actorId := trimKeyId(keyId)
	actor, err := svc.actors().FindOrCreateActor(actorId)
	if err != nil {
		return nil, fmt.Errorf("findorcreateactor: %w", err)
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
