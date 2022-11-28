package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/m/mastodon"
	"github.com/go-fed/httpsig"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FollowCmd struct {
	Object string `help:"object to follow"`
	Actor  string `help:"actor to follow with"`
}

func (f *FollowCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}
	var instance mastodon.Instance
	if err := db.First(&instance).Error; err != nil {
		return err
	}

	body, err := json.Marshal(map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       fmt.Sprintf("https://%s/%s", instance.Domain, uuid.New().String()),
		"type":     "Follow",
		"object":   f.Object,
		"actor":    f.Actor,
	})
	if err != nil {
		return err
	}
	username, domain, err := parseActor(f.Actor)
	if err != nil {
		return err
	}
	var account mastodon.Account
	if err := db.Where("username = ? AND domain = ?", username, domain).First(&account).Error; err != nil {
		return err
	}

	var actor map[string]interface{}
	if err := requests.URL(f.Actor).Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).ToJSON(&actor).Fetch(context.Background()); err != nil {
		return err
	}

	u, err := url.Parse(f.Object)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s/inbox", u.Scheme, u.Host), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/activity+json")
	sign(req, body, &account)

	if ctx.Debug {
		fmt.Printf("%s %s %s\n", req.Method, req.URL, req.Proto)
		for k := range req.Header {
			fmt.Printf("%s: %s\n", k, req.Header.Get(k))
		}
		fmt.Println()

	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if ctx.Debug {
		fmt.Printf("%s %s\n", resp.Proto, resp.Status)
		for k := range resp.Header {
			fmt.Printf("%s: %s\n", k, resp.Header.Get(k))
		}
		fmt.Println()
		io.Copy(os.Stdout, resp.Body)
		fmt.Println()
	}
	return nil
}

func sign(r *http.Request, body []byte, account *mastodon.Account) {
	r.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // Date must be in GMT, not UTC 🤯
	privPem, _ := pem.Decode(account.PrivateKey)
	if privPem.Type != "RSA PRIVATE KEY" {
		log.Fatal("expected RSA PRIVATE KEY")
	}

	var parsedKey interface{}
	var err error
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPem.Bytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPem.Bytes); err != nil { // note this returns type `interface{}`
			log.Fatal(err)
		}
	}

	var privateKey *rsa.PrivateKey
	var ok bool
	privateKey, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		log.Fatal("failed to parse RSA private key")
	}
	// The "Date" and "Digest" headers must already be set on r, as well as r.URL.
	headersToSign := []string{httpsig.RequestTarget, "date", "digest"}
	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		headersToSign,
		httpsig.Signature,
		60,
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := signer.SignRequest(privateKey, account.PublicKeyID(), r, body); err != nil {
		log.Fatal(err)
	}
}

func parseActor(acct string) (string, string, error) {
	url, err := url.Parse(acct)
	if err != nil {
		return "", "", fmt.Errorf("splitAcct: %w", err)
	}
	return path.Base(url.Path), url.Host, nil
}
