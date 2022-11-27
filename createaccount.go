package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"

	"github.com/davecheney/m/mastodon"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CreateAccountCmd struct {
	Email    string `required:"" help:"email address of the user to create"`
	Password string `required:"" help:"password of the user to create"`
	Admin    bool   `help:"create an admin account"`
}

func (c *CreateAccountCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	passwd, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	keypair, err := generateRSAKeypair()
	if err != nil {
		return err
	}

	parts := strings.Split(c.Email, "@")
	if len(parts) != 2 {
		return errors.New("invalid email address")
	}
	username := parts[0]
	domain := parts[1]

	account := &mastodon.Account{
		Username:    username,
		Domain:      domain,
		DisplayName: username,
		Email:       c.Email,
		Locked:      false,

		EncryptedPassword: passwd,
		PrivateKey:        keypair.privateKey,
		PublicKey:         keypair.publicKey,
	}
	if err := db.Create(account).Error; err != nil {
		return err
	}
	if c.Admin {
		var instance mastodon.Instance
		if err := db.Where("domain = ?", account.Domain).First(&instance).Error; err != nil {
			return err
		}
		instance.AdminAccountID = &account.ID
		return db.Save(&instance).Error
	}
	return nil
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
