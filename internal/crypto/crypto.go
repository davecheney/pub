// package crypto provides a simple interface to common cryptographic primitives.
package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// KeyPair represents a public/private keypair in PEM format.
type Keypair struct {
	PublicKey  []byte
	PrivateKey []byte
}

func GenerateRSAKeypair() (*Keypair, error) {
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
	return &Keypair{
		PublicKey:  publicKeyPem,
		PrivateKey: privateKeyPem,
	}, nil
}

// ParseRSAPrivateKey parses a PEM encoded private key, and returns
// the public key and private key.
func ParseRSAPrivateKey(pemBytes []byte) (*rsa.PublicKey, *rsa.PrivateKey, error) {
	privPem, _ := pem.Decode(pemBytes)
	if privPem == nil || privPem.Type != "RSA PRIVATE KEY" {
		return nil, nil, errors.New("expected RSA PRIVATE KEY")
	}

	var parsedKey interface{}
	var err error
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPem.Bytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPem.Bytes); err != nil { // note this returns type `interface{}`
			return nil, nil, err
		}
	}

	switch privateKey := parsedKey.(type) {
	case *rsa.PrivateKey:
		return &privateKey.PublicKey, privateKey, nil
	default:
		return nil, nil, errors.New("expected *rsa.PrivateKey")
	}
}
