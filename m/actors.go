package m

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/davecheney/m/internal/activitypub"
	"github.com/davecheney/m/internal/snowflake"
	"gorm.io/gorm"
)

type Actor struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement:false"`
	UpdatedAt      time.Time
	Type           string `gorm:"type:enum('Person', 'Application', 'Service', 'Group', 'Organization', 'LocalPerson');default:'Person';not null"`
	URI            string `gorm:"uniqueIndex;size:128"`
	Name           string `gorm:"size:64;uniqueIndex:idx_actor_name_domain"`
	Domain         string `gorm:"size:64;uniqueIndex:idx_actor_name_domain"`
	DisplayName    string `gorm:"size:128"`
	Locked         bool
	Note           string
	FollowersCount int32 `gorm:"default:0;not null"`
	FollowingCount int32 `gorm:"default:0;not null"`
	StatusesCount  int32 `gorm:"default:0;not null"`
	LastStatusAt   time.Time
	Avatar         string
	Header         string
	PublicKey      []byte   `gorm:"not null"`
	Attachments    []any    `gorm:"serializer:json"`
	Statuses       []Status `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Following      []Actor  `gorm:"many2many:account_following"`
	Followers      []Actor  `gorm:"many2many:account_followers"`
	Favourites     []Status `gorm:"many2many:favourites;"`
	Relationships  []Relationship
}

type Relationship struct {
	ActorID  uint64 `gorm:"primarykey"`
	TargetID uint64 `gorm:"primarykey"`
	Target   *Actor
	Muting   bool
	Blocking bool
}

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

func (a *Actor) PublicKeyID() string {
	return fmt.Sprintf("%s#main-key", a.URI)
}

func (a *Actor) URL() string {
	return fmt.Sprintf("https://%s/@%s", a.Domain, a.Name)
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

func (f *RemoteActorFetcher) Fetch(uri string) (*Actor, error) {
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

	return &Actor{
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
func (a *Actors) FindByURI(uri string) (*Actor, error) {
	username, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	return a.Find(username, domain)
}

func (a *Actors) Find(name, domain string) (*Actor, error) {
	var actor Actor
	err := a.service.db.Where("name = ? AND domain = ?", name, domain).First(&actor).Error
	if err != nil {
		return nil, err
	}
	return &actor, nil
}

// FindOrCreate finds an account by its URI, or creates it if it doesn't exist.
func (a *Actors) FindOrCreate(uri string, createFn func(string) (*Actor, error)) (*Actor, error) {
	name, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	var actor Actor
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
