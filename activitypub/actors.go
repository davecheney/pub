package activitypub

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"path"

	"gorm.io/gorm"
)

type Actor struct {
	gorm.Model
	ActorID   string `gorm:"uniqueIndex"`
	Type      string
	Object    []byte
	PublicKey string
}

func (a *Actor) Username() string {
	url, err := url.Parse(a.ActorID)
	if err != nil {
		panic(err)
	}
	return path.Base(url.Path)
}

func (a *Actor) Domain() string {
	url, err := url.Parse(a.ActorID)
	if err != nil {
		panic(err)
	}
	return url.Host
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
