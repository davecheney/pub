package activitypub

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
)

// Client is an ActivityPub client which can be used to fetch remote
// ActivityPub resources.
type Client struct {
	keyID      string
	privateKey crypto.PrivateKey
}

// NewClient returns a new ActivityPub client.
func NewClient(keyID string, privateKeyPem []byte) (*Client, error) {
	privPem, _ := pem.Decode(privateKeyPem)
	if privPem.Type != "RSA PRIVATE KEY" {
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
		keyID:      keyID,
		privateKey: privateKey,
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
	return fmt.Sprintf("%s %s: %d: %s: %s", e.Method, e.URI, e.StatusCode, e.err, e.Body)
}

// Get fetches the ActivityPub resource at the given URL.
func (c *Client) Get(uri string) (map[string]any, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json, application/ld+json")
	if err := signRequest(req, c.keyID, c.privateKey, nil); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	buf, _ := httputil.DumpRequest(req, false)
	fmt.Println("client#get:", string(buf))
	resp, err := http.DefaultClient.Do(req)
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
	if err := c.sign(req, body); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}
	// buf, _ := httputil.DumpRequest(req, false)
	// fmt.Println("client#post:", string(buf))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
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

func (c *Client) sign(req *http.Request, body []byte) error {
	req.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // Date must be in GMT, not UTC 🤯
	headersToSign := []string{
		httpsig.RequestTarget,
	}
	switch req.Method {
	case "GET":
		req.Header.Set("Host", req.URL.Host) // because httpsig can't do this for us
		defer req.Header.Del("Host")
		headersToSign = append(headersToSign, "host", "date", "accept")
		body = nil // httpsig uses body == nil, no len(body) == 0, so make really sure its nil
	case "POST":
		headersToSign = append(headersToSign, "date", "digest")
	}
	signer, _, err := httpsig.NewSigner(
		nil, // the default is fine for us
		httpsig.DigestSha256,
		headersToSign,
		httpsig.Signature,
		60,
	)
	if err != nil {
		return err
	}
	return signer.SignRequest(c.privateKey, c.keyID, req, body)
}

func signRequest(req *http.Request, keyID string, privateKey crypto.PrivateKey, body []byte) error {
	req.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // Date must be in GMT, not UTC 🤯
	headersToSign := []string{
		httpsig.RequestTarget,
	}
	switch req.Method {
	case "GET":
		headersToSign = append(headersToSign, "host", "date", "accept")
		body = nil // httpsig uses body == nil, no len(body) == 0, so make really sure its nil
	case "POST":
		headersToSign = append(headersToSign, "date", "digest")
	}

	var sb bytes.Buffer
	for _, header := range headersToSign {
		switch header {
		case httpsig.RequestTarget:
			sb.WriteString("(request-target): ")
			sb.WriteString(strings.ToLower(req.Method))
			sb.WriteString(" ")
			sb.WriteString(req.URL.Path)

			if req.URL.RawQuery != "" {
				sb.WriteString("?")
				sb.WriteString(req.URL.RawQuery)
			}
		case "Host", "host":
			sb.WriteString("host: ")
			sb.WriteString(req.Host)
		case "Date", "date":
			sb.WriteString("date: ")
			sb.WriteString(req.Header.Get("Date"))
		case "Accept", "accept":
			sb.WriteString("accept: ")
			sb.WriteString(req.Header.Get("Accept"))
		}
		sb.WriteString("\n")
	}
	msg := strings.TrimSpace(sb.String()) // no whitespace at the end of the signing string

	hash := sha256.New()
	hash.Write([]byte(msg))
	digest := hash.Sum(nil)

	// fmt.Printf("string to sign: %s\n", sb.String())
	// fmt.Printf("hash: %+v\n", hash)
	// fmt.Printf("digest: %x\n", digest)

	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey.(*rsa.PrivateKey), crypto.SHA256, digest)
	if err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString(sig)
	req.Header.Set("Signature", fmt.Sprintf(`keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`, keyID, strings.Join(headersToSign, " "), enc))
	return nil
}
