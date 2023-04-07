package main

import (
	"fmt"

	"github.com/davecheney/pub/internal/crypto"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CreateInstanceCmd struct {
	Domain      string `required:"" help:"domain name of the instance to create"`
	Title       string `required:"" help:"title of the instance to create"`
	Description string `required:"" help:"description of the instance to create"`
	AdminEmail  string `required:"" help:"email address of the admin account to create"`
}

func (c *CreateInstanceCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	kp, err := crypto.GenerateRSAKeypair()
	if err != nil {
		return err
	}

	// use the first 72 bytes of the private key as the bcrypt password
	passwd := trim(kp.PrivateKey, 72)

	encrypted, err := bcrypt.GenerateFromPassword(passwd, bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return withTransaction(db, func(tx *gorm.DB) error {

		admin := models.Actor{
			ID:          snowflake.Now(),
			Type:        "LocalService",
			URI:         fmt.Sprintf("https://%s/u/%s", c.Domain, "admin"),
			Name:        "admin",
			Domain:      c.Domain,
			DisplayName: "admin",
			Locked:      false,
			Note:        "The admin account for " + c.Domain,
			Avatar:      "https://avatars.githubusercontent.com/u/1024?v=4",
			Header:      "https://avatars.githubusercontent.com/u/1024?v=4",
			PublicKey:   kp.PublicKey,
		}
		if err := tx.Create(&admin).Error; err != nil {
			return err
		}

		instance := models.Instance{
			ID:               snowflake.Now(),
			Domain:           c.Domain,
			SourceURL:        "https://github.com/davecheney/pub",
			Title:            c.Title,
			ShortDescription: c.Description,
			Description:      c.Description,
			Thumbnail:        "https://avatars.githubusercontent.com/u/1024?v=4",
			Rules: []models.InstanceRule{{
				Text: "No loafing",
			}},
		}
		if err := tx.Create(&instance).Error; err != nil {
			return err
		}

		var adminRole models.AccountRole
		if err := tx.Where("name = ?", "admin").FirstOrCreate(&adminRole, models.AccountRole{
			Name:        "admin",
			Position:    1,
			Permissions: 0xFFFFFFFF,
			Highlighted: true,
		}).Error; err != nil {
			return err
		}

		adminAccount := models.Account{
			ID:                snowflake.Now(),
			InstanceID:        instance.ID,
			ActorID:           admin.ID,
			Email:             c.AdminEmail,
			EncryptedPassword: encrypted,
			PrivateKey:        kp.PrivateKey,
			RoleID:            adminRole.ID,
		}
		if err := tx.Create(&adminAccount).Error; err != nil {
			return err
		}

		return tx.Model(&instance).Update("admin_id", adminAccount.ID).Error
	})
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

func withTransaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	tx := db.Begin()
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
