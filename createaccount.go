package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
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

		keypair, err := generateRSAKeypair()
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
			PublicKey:   keypair.publicKey,
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
			PrivateKey:        keypair.privateKey,
			RoleID:            userRole.ID,
		}
		return tx.Create(&account).Error
	})

}

type keypair struct {
	publicKey  []byte
	privateKey []byte
}

func generateRSAKeypair() (*keypair, error) {
	privatekey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	publickey := &privatekey.PublicKey
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privatekey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateKeyPem := pem.EncodeToMemory(privateKeyBlock)
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publickey)
	if err != nil {
		return nil, err
	}
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyPem := pem.EncodeToMemory(publicKeyBlock)
	return &keypair{
		publicKey:  publicKeyPem,
		privateKey: privateKeyPem,
	}, nil
}
