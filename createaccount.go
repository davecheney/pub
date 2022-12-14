package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"

	"github.com/davecheney/m/m"
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

	parts := strings.Split(c.Email, "@")
	if len(parts) != 2 {
		return errors.New("invalid email address")
	}
	username := parts[0]
	domain := parts[1]

	var instance m.Instance
	if err := db.Where("domain = ?", domain).First(&instance).Error; err != nil {
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

	actor := &m.Actor{
		Name:        username,
		Domain:      domain,
		Type:        "LocalPerson",
		DisplayName: username,
		PublicKey:   keypair.publicKey,
		Locked:      false,
	}
	if err := db.Create(actor).Error; err != nil {
		return err
	}

	account := &m.Account{
		ActorID:           actor.ID,
		Actor:             actor,
		Email:             c.Email,
		EncryptedPassword: passwd,
		PrivateKey:        keypair.privateKey,
	}
	if err := db.Model(&instance).Association("Accounts").Append(account); err != nil {
		return err
	}
	if c.Admin {
		instance.AdminID = &account.ID
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
