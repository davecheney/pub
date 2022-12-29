package m

import (
	"time"

	"gorm.io/gorm"
)

// An Instance is an ActivityPub domain managed by this server.
// An Instance has many Accounts.
// An Instance has many Applications.
// An Instance has many InstanceRules.
// An Instance has an Admin Account.
type Instance struct {
	ID               uint32 `gorm:"primarykey"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Domain           string `gorm:"size:64;uniqueIndex"`
	AdminID          *uint32
	Admin            *Account
	SourceURL        string
	Title            string `gorm:"size:64"`
	ShortDescription string
	Description      string
	Thumbnail        string `gorm:"size:64"`
	AccountsCount    int    `gorm:"default:0;not null"`
	StatusesCount    int    `gorm:"default:0;not null"`

	DomainsCount int `gorm:"-"`

	Rules        []InstanceRule `gorm:"foreignKey:InstanceID"`
	Applications []Application
}

func (i *Instance) AfterCreate(tx *gorm.DB) error {
	return i.updateAccountsCount(tx)
}

func (i *Instance) updateAccountsCount(tx *gorm.DB) error {
	var count int64
	err := tx.Model(&Account{}).Where("instance_id = ?", i.ID).Count(&count).Error
	if err != nil {
		return err
	}
	return tx.Model(i).Update("accounts_count", count).Error
}

func (i *Instance) updateStatusesCount(tx *gorm.DB) error {
	var count int64
	err := tx.Model(&Status{}).Joins("Account").Where("instance_id = ?", i.ID).Count(&count).Error
	if err != nil {
		return err
	}
	return tx.Model(i).Update("statuses_count", count).Error
}

// An Application is a registered client application.
// An Application belongs to an Instance.
// An Application has many Tokens.
type Application struct {
	ID           uint32 `gorm:"primarykey"`
	CreatedAt    time.Time
	InstanceID   uint32
	Instance     *Instance
	Name         string  `gorm:"size:64;not null"`
	Website      *string `gorm:"size:64"`
	RedirectURI  string  `gorm:"size:128;not null"`
	ClientID     string  `gorm:"size:64;not null"`
	ClientSecret string  `gorm:"size:64;not null"`
	VapidKey     string  `gorm:"size:128;not null"`
	Tokens       []Token
}

// A Token is an access token for an Application.
// A Token belongs to an Account.
type Token struct {
	AccessToken       string `gorm:"size:64;primaryKey"`
	CreatedAt         time.Time
	AccountID         uint32
	Account           *Account
	ApplicationID     uint32
	TokenType         string `gorm:"type:enum('Bearer');not null"`
	Scope             string `gorm:"size:64;not null"`
	AuthorizationCode string `gorm:"size:64;not null"`
}

// An Account is a user account on an Instance.
// An Account belongs to an Actor.
type Account struct {
	ID                uint32 `gorm:"primarykey"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	InstanceID        uint32 `gorm:"index"`
	ActorID           uint64
	Actor             *Actor
	Notifications     []Notification
	Lists             []AccountList
	Email             string `gorm:"size:64;not null"`
	EncryptedPassword []byte `gorm:"size:60;not null"`
	PrivateKey        []byte `gorm:"not null"`
	ClientFilters     []ClientFilter
	RoleID            uint32
	Role              *AccountRole
}

type AccountRole struct {
	ID          uint32 `gorm:"primarykey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string `gorm:"size:16;not null"`
	Color       string `gorm:"size:8;not null,default:''"`
	Position    int32
	Permissions uint32
	Highlighted bool
}

// https://docs.joinmastodon.org/entities/V1_Filter/
type ClientFilter struct {
	gorm.Model
	AccountID    uint32
	Phrase       string
	WholeWord    bool
	Context      []string `gorm:"serializer:json"`
	ExpiresAt    time.Time
	Irreversible bool
}
