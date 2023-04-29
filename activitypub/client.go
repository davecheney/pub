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
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/pub/internal/httpsig"
	"github.com/davecheney/pub/models"
	"github.com/google/uuid"
)

// Client is an ActivityPub client which can be used to fetch remote
// ActivityPub resources.
type Client struct {
	keyID      string
	privateKey crypto.PrivateKey
	ctx        context.Context
}

// NewClient returns a new ActivityPub client.
func NewClient(ctx context.Context, signAs *models.Account) (*Client, error) {
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
		ctx:        ctx,
	}, nil
}

type Error struct {
	StatusCode int
	URI        string
	Method     string
	Body       string
	err        error
}

func (e *Error) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Method)
	sb.WriteString(" ")
	sb.WriteString(e.URI)
	sb.WriteString(": ")
	fmt.Fprintf(&sb, "%d ", e.StatusCode)
	if e.err != nil {
		sb.WriteString(e.err.Error())
		sb.WriteString(": ")
	}
	sb.WriteString(e.Body)
	return sb.String()
}

// Follow sends a follow request from the Account to the Target Actor's inbox.
func Follow(ctx context.Context, follower *models.Account, target *models.Actor) error {
	inbox := target.Inbox()
	if inbox == "" {
		return fmt.Errorf("no inbox found for %s", target.URI)
	}
	c, err := NewClient(ctx, follower)
	if err != nil {
		return err
	}
	return c.Post(inbox, map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       uuid.New().String(),
		"type":     "Follow",
		"object":   target.URI,
		"actor":    follower.Actor.URI,
	})
}

// Unfollow sends an unfollow request from the Account to the Target Actor's inbox.
func Unfollow(ctx context.Context, follower *models.Account, target *models.Actor) error {
	inbox := target.Inbox()
	if inbox == "" {
		return fmt.Errorf("no inbox found for %s", target.URI)
	}
	c, err := NewClient(ctx, follower)
	if err != nil {
		return err
	}
	return c.Post(inbox, map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       uuid.New().String(),
		"type":     "Undo",
		"object": map[string]any{
			"type":   "Follow",
			"object": target.URI,
			"actor":  follower.Actor.URI,
		},
		"actor": follower.Actor.URI,
	})
}

// Like sends a like request from the Account to the Statuses Actor's inbox.
func Like(ctx context.Context, liker *models.Account, target *models.Status) error {
	inbox := target.Actor.Inbox()
	if inbox == "" {
		return fmt.Errorf("no inbox found for %s", target.Actor.URI)
	}
	c, err := NewClient(ctx, liker)
	if err != nil {
		return err
	}
	return c.Post(inbox, map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       uuid.New().String(),
		"type":     "Like",
		"object":   target.URI,
		"actor":    liker.Actor.URI,
	})
}

// Unlike sends an undo like request from the Account to the Statuses Actor's inbox.
func Unlike(ctx context.Context, liker *models.Account, target *models.Status) error {
	inbox := target.Actor.Inbox()
	if inbox == "" {
		return fmt.Errorf("no inbox found for %s", target.Actor.URI)
	}
	c, err := NewClient(ctx, liker)
	if err != nil {
		return err
	}
	return c.Post(inbox, map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       uuid.New().String(),
		"type":     "Undo",
		"object": map[string]any{
			"type":   "Like",
			"object": target.URI,
			"actor":  liker.Actor.URI,
		},
		"actor": liker.Actor.URI,
	})
}

// FetchActor fetches the Actor at the given URI.
func FetchActor(ctx context.Context, signer *models.Account, uri string) (*Actor, error) {
	c, err := NewClient(ctx, signer)
	if err != nil {
		return nil, err
	}
	var actor Actor
	return &actor, c.Fetch(uri, &actor)
}

// FetchStatus fetches the Status at the given URI.
func FetchStatus(ctx context.Context, signer *models.Account, uri string) (*Status, error) {
	c, err := NewClient(ctx, signer)
	if err != nil {
		return nil, err
	}
	var status Status
	return &status, c.Fetch(uri, &status)
}

// Fetch fetches the ActivityPub resource at the given URL and decodes it into the given object.
func (c *Client) Fetch(uri string, obj interface{}) error {
	return requests.URL(uri).
		Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
		Transport(c).
		CheckContentType("application/ld+json", "application/activity+json", "application/json").
		CheckStatus(http.StatusOK).
		ToJSON(obj).
		Fetch(c.ctx)
}

func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := httpsig.Sign(req, c.keyID, c.privateKey, nil); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// Post posts the given ActivityPub object to the given URL.
func (c *Client) Post(url string, obj map[string]any) error {
	return requests.URL(url).
		Header("Content-Type", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
		BodyJSON(obj).
		Transport(c).
		CheckStatus(http.StatusOK, http.StatusCreated).
		Fetch(c.ctx)
}
