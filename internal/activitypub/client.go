package activitypub

import (
	"context"
	"crypto"
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/pub/internal/httpsig"
	"github.com/go-json-experiment/json"
)

// ClientKey is the key used to store the ActivityPub client in the context.
var ClientKey = struct{}{}

// FromContext returns the ActivityPub client from the given context.
func FromContext(ctx context.Context) (*Client, bool) {
	c, ok := ctx.Value(ClientKey).(*Client)
	return c, ok
}

// WithClient returns a new context with the given ActivityPub client.
func WithClient(ctx context.Context, c *Client) context.Context {
	return context.WithValue(ctx, ClientKey, c)
}

// Client is an ActivityPub client which can be used to fetch remote
// ActivityPub resources.
type Client struct {
	keyID      string
	privateKey crypto.PrivateKey
}

// Signer represents an object that can sign HTTP requests.
type Signer interface {
	PublicKeyID() string
	PrivKey() (*rsa.PrivateKey, error)
}

// NewClient returns a new ActivityPub client.
func NewClient(signAs Signer) (*Client, error) {
	privateKey, err := signAs.PrivKey()
	if err != nil {
		return nil, err
	}
	return &Client{
		keyID:      signAs.PublicKeyID(),
		privateKey: privateKey,
	}, nil
}

// Fetch fetches the ActivityPub resource at the given URL and decodes it into the given object.
func (c *Client) Fetch(ctx context.Context, uri string, obj interface{}) error {
	return requests.URL(uri).
		Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
		Transport(requests.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if err := httpsig.Sign(req, c.keyID, c.privateKey, nil); err != nil {
				return nil, fmt.Errorf("failed to sign request: %w", err)
			}
			return http.DefaultTransport.RoundTrip(req)
		})).
		CheckContentType(
			"application/ld+json",
			"application/activity+json",
			"application/json",
			"application/octet-stream", // sigh
		).
		CheckStatus(http.StatusOK).
		ToJSON(obj).
		Fetch(ctx)
}

// Post posts the given ActivityPub object to the given URL.
func (c *Client) Post(ctx context.Context, url string, obj map[string]any) error {
	body, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return requests.URL(url).
		BodyBytes(body).
		Header("Content-Type", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
		Transport(requests.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if err := httpsig.Sign(req, c.keyID, c.privateKey, body); err != nil {
				return nil, fmt.Errorf("failed to sign request: %w", err)
			}
			return http.DefaultTransport.RoundTrip(req)
		})).
		ToWriter(os.Stderr).
		CheckStatus(http.StatusOK, http.StatusCreated, http.StatusAccepted).
		Fetch(ctx)
}
