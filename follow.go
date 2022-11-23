package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/google/uuid"
)

type FollowCmd struct {
	Object string `help:"object to follow"`
	Actor  string `help:"actor to follow with"`
}

func (f *FollowCmd) Run(ctx *Context) error {
	body, err := json.Marshal(map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       fmt.Sprintf("https://cheney.net/%s", uuid.New().String()),
		"type":     "Follow",
		"object":   f.Object,
		"actor":    f.Actor,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", f.Actor, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetchActor: status: %d", resp.StatusCode)
	}
	var actor map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return fmt.Errorf("fetchActor: jsondecode: %w", err)
	}

	u, err := url.Parse(f.Object)
	if err != nil {
		return err
	}
	req, err = http.NewRequest("POST", fmt.Sprintf("%s://%s/inbox", u.Scheme, u.Host), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/activity+json")
	sign(req, body, actor["publicKey"].(map[string]interface{})["id"].(string))

	if ctx.Debug {
		fmt.Printf("%s %s %s\n", req.Method, req.URL, req.Proto)
		for k := range req.Header {
			fmt.Printf("%s: %s\n", k, req.Header.Get(k))
		}
		fmt.Println()

	}

	resp, err = http.DefaultClient.Do(req)
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

func sign(r *http.Request, body []byte, pubKeyId string) {
	r.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // Date must be in GMT, not UTC 🤯
	priv, err := ioutil.ReadFile("private.pem")
	if err != nil {
		log.Fatal(err)
	}
	privPem, _ := pem.Decode(priv)
	var privPemBytes []byte
	if privPem.Type != "RSA PRIVATE KEY" {
		log.Fatal("expected RSA PRIVATE KEY")
	}

	privPemBytes = privPem.Bytes

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err != nil { // note this returns type `interface{}`
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
	if err := signer.SignRequest(privateKey, pubKeyId, r, body); err != nil {
		log.Fatal(err)
	}
}
