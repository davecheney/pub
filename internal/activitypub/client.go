package activitypub

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/davecheney/pub/internal/httpsig"
	"github.com/davecheney/pub/models"
	"github.com/go-json-experiment/json"
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

// Get fetches the ActivityPub resource at the given URL.
func (c *Client) Get(uri string) (map[string]any, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json, application/ld+json")
	if err := httpsig.Sign(req, c.keyID, c.privateKey, nil); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(c.ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return c.bodyToObj(resp)
}

// Post posts the given ActivityPub object to the given URL.
func (c *Client) Post(url string, obj map[string]any) error {
	body, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/activity+json")
	if err := httpsig.Sign(req, c.keyID, c.privateKey, body); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(c.ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return &Error{
			StatusCode: resp.StatusCode,
			URI:        resp.Request.URL.String(),
			Method:     resp.Request.Method,
			Body:       string(body),
		}
	}
	return nil
}

// bodyToObj reads the body of the given response and returns the
// ActivityPub object as a map[string]any.
func (c *Client) bodyToObj(resp *http.Response) (map[string]any, error) {
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &Error{
			StatusCode: resp.StatusCode,
			URI:        resp.Request.URL.String(),
			Method:     resp.Request.Method,
			Body:       string(body),
		}
	}
	var obj map[string]any
	if err := json.UnmarshalFull(resp.Body, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}
