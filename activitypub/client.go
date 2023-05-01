package activitypub

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/pub/internal/httpsig"
	"github.com/davecheney/pub/models"
)

// Client is an ActivityPub client which can be used to fetch remote
// ActivityPub resources.
type Client struct {
	keyID      string
	privateKey crypto.PrivateKey
}

// NewClient returns a new ActivityPub client.
func NewClient(signAs *models.Account) (*Client, error) {
	privPem, _ := pem.Decode(signAs.PrivateKey)
	if privPem == nil || privPem.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("expected RSA PRIVATE KEY")
	}

	var parsedKey interface{}
	var err error
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPem.Bytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPem.Bytes); err != nil { // note this returns type `interface{}`
			return nil, err
		}
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("expected *rsa.PrivateKey")
	}

	return &Client{
		keyID:      signAs.Actor.PublicKeyID(),
		privateKey: privateKey,
	}, nil
}

// Fetch fetches the ActivityPub resource at the given URL and decodes it into the given object.
func (c *Client) Fetch(ctx context.Context, uri string, obj interface{}) error {
	return requests.URL(uri).
		Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
		Transport(c).
		CheckContentType("application/ld+json", "application/activity+json", "application/json").
		CheckStatus(http.StatusOK).
		ToJSON(obj).
		Fetch(ctx)
}

func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := httpsig.Sign(req, c.keyID, c.privateKey, nil); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// Post posts the given ActivityPub object to the given URL.
func (c *Client) Post(ctx context.Context, url string, obj map[string]any) error {
	return requests.URL(url).
		Header("Content-Type", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
		BodyJSON(obj).
		Transport(c).
		CheckStatus(http.StatusOK, http.StatusCreated).
		Fetch(ctx)
}
