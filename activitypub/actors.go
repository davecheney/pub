package activitypub

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"path"

	"github.com/carlmjohnson/requests"
	"gorm.io/gorm"
)

type Actor struct {
	gorm.Model
	ActorID   string `gorm:"uniqueIndex"`
	Type      string
	Object    map[string]interface{} `gorm:"serializer:json"`
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

type Actors struct {
	db *gorm.DB
}

func NewActors(db *gorm.DB) *Actors {
	return &Actors{
		db: db,
	}
}

// FindOrCreateActor finds an actor by id or creates a new one.
func (a *Actors) FindOrCreateActor(id string) (*Actor, error) {
	var actor Actor
	err := a.db.Where("actor_id = ?", id).First(&actor).Error
	if err == nil {
		// found cached key
		return &actor, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var obj map[string]interface{}
	if err := requests.URL(id).Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).ToJSON(&obj).Fetch(context.Background()); err != nil {
		return nil, err
	}

	actor = Actor{
		ActorID:   id,
		Type:      obj["type"].(string),
		Object:    obj,
		PublicKey: obj["publicKey"].(map[string]interface{})["publicKeyPem"].(string),
	}
	if err := a.db.Create(&actor).Error; err != nil {
		return nil, fmt.Errorf("fetchActor: create: %w", err)
	}
	return &actor, nil
}
