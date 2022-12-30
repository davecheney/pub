package m

import (
	"errors"
	"net/url"
	"time"

	"github.com/davecheney/m/internal/activitypub"
	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/internal/snowflake"
	"gorm.io/gorm"
)

type Webfinger struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	ActorID   uint64
	Webfinger struct {
		Subject string   `json:"subject"`
		Aliases []string `json:"aliases"`
		Links   []struct {
			Rel      string `json:"rel"`
			Type     string `json:"type"`
			Href     string `json:"href"`
			Template string `json:"template"`
		} `json:"links"`
	} `gorm:"serializer:json"`
}

type Actors struct {
	service *Service
}

func (a *Actors) NewRemoteActorFetcher() *RemoteActorFetcher {
	return &RemoteActorFetcher{
		service: a.service,
	}
}

type RemoteActorFetcher struct {
	service *Service
}

func (f *RemoteActorFetcher) Fetch(uri string) (*models.Actor, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	obj, err := f.fetch(uri)
	if err != nil {
		return nil, err
	}

	published := timeFromAny(obj["published"])
	if published.IsZero() {
		published = time.Now()
	}

	return &models.Actor{
		ID:           snowflake.TimeToID(published),
		Type:         stringFromAny(obj["type"]),
		Name:         stringFromAny(obj["preferredUsername"]),
		Domain:       u.Host,
		URI:          stringFromAny(obj["id"]),
		DisplayName:  stringFromAny(obj["name"]),
		Locked:       boolFromAny(obj["manuallyApprovesFollowers"]),
		Note:         stringFromAny(obj["summary"]),
		Avatar:       stringFromAny(mapFromAny(obj["icon"])["url"]),
		Header:       stringFromAny(mapFromAny(obj["image"])["url"]),
		LastStatusAt: time.Now(),
		Attachments:  anyToSlice(obj["attachment"]),
		PublicKey:    []byte(stringFromAny(mapFromAny(obj["publicKey"])["publicKeyPem"])),
	}, nil
}

func (f *RemoteActorFetcher) fetch(uri string) (map[string]any, error) {
	// use admin account to sign the request
	signAs, err := f.service.Accounts().FindAdminAccount()
	if err != nil {
		return nil, err
	}
	c, err := activitypub.NewClient(signAs.Actor.PublicKeyID(), signAs.PrivateKey)
	if err != nil {
		return nil, err
	}
	return c.Get(uri)
}

// FindByURI returns an account by its URI if it exists locally.
func (a *Actors) FindByURI(uri string) (*models.Actor, error) {
	username, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	return a.Find(username, domain)
}

func (a *Actors) Find(name, domain string) (*models.Actor, error) {
	var actor models.Actor
	err := a.service.db.Where("name = ? AND domain = ?", name, domain).First(&actor).Error
	if err != nil {
		return nil, err
	}
	return &actor, nil
}

// FindOrCreate finds an account by its URI, or creates it if it doesn't exist.
func (a *Actors) FindOrCreate(uri string, createFn func(string) (*models.Actor, error)) (*models.Actor, error) {
	name, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	var actor models.Actor
	err = a.service.db.Where("name = ? AND domain = ?", name, domain).First(&actor).Error
	if err == nil {
		// found cached key
		return &actor, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	acc, err := createFn(uri)
	if err != nil {
		return nil, err
	}
	err = a.service.db.Create(acc).Error
	return acc, err
}
