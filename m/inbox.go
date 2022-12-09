package m

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Activity struct {
	gorm.Model
	Object map[string]interface{} `gorm:"serializer:json"`
}

func (Activity) TableName() string {
	return "inbox"
}

type Inboxes struct {
	service *Service
}

func (i *Inboxes) Create(w http.ResponseWriter, r *http.Request) {
	if err := i.validateSignature(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var body map[string]interface{}
	if err := json.UnmarshalFull(r.Body, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// buf, _ := httputil.DumpRequest(r, false)
	// fmt.Println("inbox#create:", string(buf))

	activity := Activity{
		Object: body,
	}
	if err := i.service.db.Create(&activity).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (i *Inboxes) validateSignature(r *http.Request) error {
	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		fmt.Println("NewVerifier:", err)
		return err
	}
	pubKey, err := i.getKey(verifier.KeyId())
	if err != nil {
		fmt.Println("getKey failed for key id:", verifier.KeyId(), err)
		return err
	}
	if err := verifier.Verify(pubKey, httpsig.RSA_SHA256); err != nil {
		fmt.Println("verify:", err)
		return err
	}
	return nil

}

func (i *Inboxes) getKey(keyId string) (crypto.PublicKey, error) {
	actorId := trimKeyId(keyId)
	fetcher := i.service.Accounts().NewRemoteAccountFetcher()
	account, err := i.service.Accounts().FindOrCreate(actorId, fetcher.Fetch)
	if err != nil {
		return nil, err
	}
	return pemToPublicKey(account.PublicKey)
}

func pemToPublicKey(key []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("pemToPublicKey: invalid pem type: %s", block.Type)
	}
	var publicKey interface{}
	var err error
	if publicKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("pemToPublicKey: parsepkixpublickey: %w", err)
	}
	return publicKey, nil
}

// trimKeyId removes the #main-key suffix from the key id.
func trimKeyId(id string) string {
	if i := strings.Index(id, "#"); i != -1 {
		return id[:i]
	}
	return id
}
