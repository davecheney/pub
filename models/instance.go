package models

import (
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/crypto"
	"github.com/davecheney/pub/internal/snowflake"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// An Instance is an ActivityPub domain managed by this server.
// An Instance has many InstanceRules.
// An Instance has one Admin Account.
type Instance struct {
	snowflake.ID     `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt        time.Time
	Domain           string `gorm:"size:64;uniqueIndex"`
	AdminID          *snowflake.ID
	Admin            *Account `gorm:"foreignKey:AdminID;constraint:OnDelete:CASCADE;<-:create;"` // the admin account for this instance
	SourceURL        string
	Title            string `gorm:"size:64"`
	ShortDescription string
	Description      string
	Thumbnail        string         `gorm:"size:64"`
	AccountsCount    int            `gorm:"default:0;not null"`
	StatusesCount    int            `gorm:"default:0;not null"`
	DomainsCount     int32          `gorm:"default:0;not null"`
	Rules            []InstanceRule `gorm:"constraint:OnDelete:CASCADE;"`
}

type InstanceRule struct {
	ID         uint32 `gorm:"primarykey"`
	InstanceID uint64
	Text       string
}

type Instances struct {
	db *gorm.DB
}

func NewInstances(db *gorm.DB) *Instances {
	return &Instances{db: db}
}

// Create creates a new instance, complete with an admin account.
func (i *Instances) Create(domain, title, description, adminEmail string) (*Instance, error) {
	var instance Instance
	err := i.db.Transaction(func(tx *gorm.DB) error {

		kp, err := crypto.GenerateRSAKeypair()
		if err != nil {
			return err
		}

		// use the first 72 bytes of the private key as the bcrypt password for the admin account
		passwd := trim(kp.PrivateKey, 72)

		encrypted, err := bcrypt.GenerateFromPassword(passwd, bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		instance = Instance{
			ID:               snowflake.Now(),
			Domain:           domain,
			SourceURL:        "https://github.com/davecheney/pub",
			Title:            title,
			ShortDescription: description,
			Description:      description,
			Thumbnail:        "https://avatars.githubusercontent.com/u/1024?v=4",
			Rules: []InstanceRule{{
				Text: "No loafing",
			}},
		}
		if err := tx.Create(&instance).Error; err != nil {
			return err
		}

		var adminRole AccountRole
		if err := tx.Where("name = ?", "admin").FirstOrCreate(&adminRole, AccountRole{
			Name:        "admin",
			Position:    1,
			Permissions: 0xFFFFFFFF,
			Highlighted: true,
		}).Error; err != nil {
			return err
		}

		adminAccount := Account{
			ID:       snowflake.Now(),
			Instance: &instance,
			Actor: &Actor{
				ID:          snowflake.Now(),
				Type:        "LocalService",
				URI:         fmt.Sprintf("https://%s/u/%s", domain, "admin"),
				Name:        "admin",
				Domain:      instance.Domain,
				DisplayName: "admin",
				Locked:      false,
				Note:        "The admin account for " + domain,
				Avatar:      "https://avatars.githubusercontent.com/u/1024?v=4",
				Header:      "https://avatars.githubusercontent.com/u/1024?v=4",
				PublicKey:   kp.PublicKey,
			},
			Email:             adminEmail,
			EncryptedPassword: encrypted,
			PrivateKey:        kp.PrivateKey,
			RoleID:            adminRole.ID,
		}
		if err := tx.Create(&adminAccount).Error; err != nil {
			return err
		}
		return tx.Model(&instance).Update("admin_id", adminAccount.ID).Error
	})
	return &instance, err
}

// trim trims the first n bytes from the given byte slice
func trim[S []T, T any](s S, n int) S {
	return s[:min(len(s), n)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
