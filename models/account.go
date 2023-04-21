package models

import (
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/crypto"
	"github.com/davecheney/pub/internal/snowflake"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// An Account is a user account on an Instance.
// An Account belongs to an Actor.
// An Account belongs to an Instance.
type Account struct {
	snowflake.ID      `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt         time.Time
	InstanceID        snowflake.ID
	Instance          *Instance `gorm:"<-:create;"`
	ActorID           snowflake.ID
	Actor             *Actor          `gorm:"<-:create;"`
	Lists             []AccountList   `gorm:"constraint:OnDelete:CASCADE;"`
	Markers           []AccountMarker `gorm:"constraint:OnDelete:CASCADE;"`
	Email             string          `gorm:"size:64;not null"`
	EncryptedPassword []byte          `gorm:"size:60;not null"`
	PrivateKey        []byte          `gorm:"not null"`
	RoleID            uint32
	Role              *AccountRole
}

func (a *Account) Name() string {
	return a.Actor.Name
}

func (a *Account) Domain() string {
	return a.Actor.Domain
}

type AccountRole struct {
	ID          uint32 `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string `gorm:"size:16;not null"`
	Color       string `gorm:"size:8;not null,default:''"`
	Position    int32
	Permissions uint32
	Highlighted bool
}

type AccountList struct {
	snowflake.ID  `gorm:"primarykey;autoIncrement:false"`
	AccountID     snowflake.ID        `gorm:"not null;"`
	Title         string              `gorm:"size:64"`
	RepliesPolicy string              `gorm:"enum('public','followers','none');not null;default:'public'"`
	Members       []AccountListMember `gorm:"constraint:OnDelete:CASCADE;"`
}

type AccountListMember struct {
	AccountListID snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	MemberID      snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	Member        *Actor       `gorm:"constraint:OnDelete:CASCADE;"`
}

type AccountMarker struct {
	ID         uint32 `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	AccountID  snowflake.ID `gorm:"not null;uniqueIndex:idx_account_name;index"`
	Name       string       `gorm:"enum('home','notifications');not null;uniqueIndex:idx_account_name"`
	Version    int32        `gorm:"not null;"`
	LastReadID snowflake.ID `gorm:"not null;"`
}

type Accounts struct {
	db *gorm.DB
}

func NewAccounts(db *gorm.DB) *Accounts {
	return &Accounts{db: db}
}

func (a *Accounts) AccountForActor(actor *Actor) (*Account, error) {
	var account Account
	if err := a.db.Joins("Actor").First(&account, "actor_id = ?", actor.ID).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (a *Accounts) Create(instance *Instance, name, email, password string) (*Account, error) {
	var account Account
	err := a.db.Transaction(func(tx *gorm.DB) error {

		keypair, err := crypto.GenerateRSAKeypair()
		if err != nil {
			return err
		}

		passwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		var userRole AccountRole
		if err := tx.Where("name = ?", "admin").FirstOrCreate(&userRole, AccountRole{
			Name:        "user",
			Position:    10,
			Permissions: 65535,
		}).Error; err != nil {
			return err
		}

		account = Account{
			ID:       snowflake.Now(),
			Instance: instance,
			Actor: &Actor{
				ID:          snowflake.Now(),
				Name:        name,
				Domain:      instance.Domain,
				URI:         fmt.Sprintf("https://%s/u/%s", instance.Domain, name),
				Type:        "LocalPerson",
				DisplayName: name,
				Avatar:      "https://avatars.githubusercontent.com/u/1024?v=4",
				Header:      "https://avatars.githubusercontent.com/u/1024?v=4",
				PublicKey:   keypair.PublicKey,
			},
			Email:             email,
			EncryptedPassword: passwd,
			PrivateKey:        keypair.PrivateKey,
			RoleID:            userRole.ID,
		}
		return tx.Create(&account).Error
	})
	return &account, err
}

type AccountPreferences struct {
	AccountID                snowflake.ID `gorm:"primarykey;autoIncrement:false"`
	PostingDefaultVisibility string       `gorm:"enum('public', 'unlisted', 'private', 'direct');not null;default:'public'"`
	PostingDefaultSensitive  bool         `gorm:"not null;default:false"`
	PostingDefaultLanguage   string       `gorm:"size:8;"`
	ReadingExpandMedia       string       `gorm:"enum('default','show_all','hide_all');not null;default:'default'"`
	ReadingExpandSpoilers    bool         `gorm:"not null;default:false"`
}
