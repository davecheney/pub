package main

import (
	"fmt"

	"github.com/davecheney/pub/internal/crypto"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CreateAccountCmd struct {
	Name     string `required:"" help:"name of the account to create"`
	Domain   string `required:"" help:"domain of the account to create"`
	Email    string `required:"" help:"email address of the account to create"`
	Password string `required:"" help:"password of the account to create"`
}

func (c *CreateAccountCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	return withTransaction(db, func(tx *gorm.DB) error {

		var instance models.Instance
		if err := tx.Where("domain = ?", c.Domain).First(&instance).Error; err != nil {
			return err
		}

		keypair, err := crypto.GenerateRSAKeypair()
		if err != nil {
			return err
		}

		passwd, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		actor := models.Actor{
			ID:          snowflake.Now(),
			Name:        c.Name,
			Domain:      c.Domain,
			URI:         fmt.Sprintf("https://%s/u/%s", c.Domain, c.Name),
			Type:        "LocalPerson",
			DisplayName: c.Name,
			Avatar:      "https://avatars.githubusercontent.com/u/1024?v=4",
			Header:      "https://avatars.githubusercontent.com/u/1024?v=4",
			PublicKey:   keypair.PublicKey,
		}
		if err := tx.Create(&actor).Error; err != nil {
			return err
		}

		var userRole models.AccountRole
		if err := tx.Where("name = ?", "admin").FirstOrCreate(&userRole, models.AccountRole{
			Name:        "user",
			Position:    10,
			Permissions: 65535,
		}).Error; err != nil {
			return err
		}

		account := models.Account{
			ID:                snowflake.Now(),
			InstanceID:        instance.ID,
			ActorID:           actor.ID,
			Email:             c.Email,
			EncryptedPassword: passwd,
			PrivateKey:        keypair.PrivateKey,
			RoleID:            userRole.ID,
		}
		return tx.Create(&account).Error
	})

}
