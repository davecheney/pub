package activitypub

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"gorm.io/gorm"
)

type Actor struct {
	gorm.Model
	ActorID   string
	Type      string
	Object    []byte
	PublicKey string
}

func (Actor) TableName() string {
	return "activitypub_actors"
}

// pemToPublicKey converts a PEM encoded public key to a crypto.PublicKey.
func (a *Actor) pemToPublicKey() (crypto.PublicKey, error) {
	block, _ := pem.Decode([]byte(a.PublicKey))
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
