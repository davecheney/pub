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
	ID               uint `gorm:"primarykey"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Domain           string `gorm:"size:64;uniqueIndex"`
	AdminID          *uint
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
	Accounts     []Account
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
	gorm.Model
	InstanceID   uint
	Instance     *Instance
	Name         string
	Website      *string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	VapidKey     string
	Tokens       []Token
}

// A Token is an access token for an Application.
// A Token belongs to an Account.
type Token struct {
	ID                uint `gorm:"primarykey"`
	CreatedAt         time.Time
	AccountID         uint
	Account           *Account
	ApplicationID     uint
	AccessToken       string
	TokenType         string
	Scope             string
	AuthorizationCode string
}

// An Account is a user account on an Instance.
// An Account belongs to an Actor.
type Account struct {
	ID                uint `gorm:"primarykey"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	InstanceID        uint
	ActorID           uint64
	Actor             *Actor
	Notifications     []Notification
	Markers           []Marker
	Lists             []AccountList
	Email             string
	EncryptedPassword []byte
	PrivateKey        []byte `gorm:"not null"`
	Tokens            []Token
	ClientFilters     []ClientFilter
	RoleID            uint
	Role              *AccountRole
}

type AccountRole struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	Color       string
	Position    int
	Permissions uint
	Highlighted bool
}

// https://docs.joinmastodon.org/entities/V1_Filter/
type ClientFilter struct {
	gorm.Model
	AccountID    uint
	Phrase       string
	WholeWord    bool
	Context      []string `gorm:"serializer:json"`
	ExpiresAt    time.Time
	Irreversible bool
}
