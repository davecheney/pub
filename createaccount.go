package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/davecheney/m/mastodon"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CreateAccountCmd struct {
	Username string `required:"" help:"username of the user to create"`
	Domain   string `required:"" help:"domain name of the instance to create"`
	Password string `required:"" help:"password of the user to create"`
}

func (c *CreateAccountCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}
	var instance mastodon.Instance
	if err := db.Where("domain = ?", c.Domain).First(&instance).Error; err != nil {
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

	account := &mastodon.Account{
		Username:    c.Username,
		Domain:      instance.Domain,
		DisplayName: c.Username,
		Locked:      false,

		EncryptedPassword: passwd,
		PrivateKey:        keypair.privateKey,
		PublicKey:         keypair.publicKey,
	}
	return db.Create(account).Error

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
