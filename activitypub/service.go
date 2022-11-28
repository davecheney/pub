package activitypub

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/davecheney/m/mastodon"
	"github.com/go-fed/httpsig"
	"gorm.io/gorm"
)

// Service implements a Mastodon service.
type Service struct {
	db       *gorm.DB
	instance *mastodon.Instance
}

// NewService returns a new instance of Service.
func NewService(db *gorm.DB, instance *mastodon.Instance) *Service {
	return &Service{
		db:       db,
		instance: instance,
	}
}

func (svc *Service) accounts() *mastodon.Accounts {
	return mastodon.NewAccounts(svc.db, svc.instance)
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
	account, err := svc.accounts().FindOrCreateAccount(actorId)
	if err != nil {
		return nil, err
	}
	return pemToPublicKey([]byte(account.PublicKey))
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
