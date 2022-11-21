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
	"os"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/google/uuid"
)

func main() {
	body, err := json.Marshal(map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       fmt.Sprintf("https://cheney.net/%s", uuid.New().String()),
		"type":     "Follow",
		// "object":   "https://hachyderm.io/users/youngnick",
		// "object": "https://mastadon.social/users/effinbirds",
		"object": "https://infosec.exchange/users/riskybusiness",
		// "object": "https://bitbang.social/users/NanoRaptor",
		// "object": "https://mas.to/users/TechConnectify",
		// "object": "https://oldbytes.space/users/48kRAM",
		// "object": "https://cheney.net/users/dave",
		"actor": "https://cheney.net/users/dave",
	})
	if err != nil {
		log.Fatal(err)
	}
	req, err := http.NewRequest("POST", "https://infosec.exchange/inbox", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/activity+json")
	sign(req, body, "https://cheney.net/users/dave#main-key2")

	dumpRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	dumpResponse(resp)
}

func sign(r *http.Request, body []byte, pubKeyId string) {
	r.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // Date must be in GMT, not UTC 🤯
	// r.Header.Set("Digest", fmt.Sprintf("SHA-256=%s", digest(sha256.New(), body)))
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

	// input := fmt.Sprintf("(request-target): %s %s\ndigest: %s\ndate: %s\n", r.Method, r.URL.Path, r.Header.Get("Digest"), r.Header.Get("Date"))
	// // Before signing, we need to hash our message
	// // The hash is what we actually sign
	// hash := sha256.New()
	// hash.Write([]byte(input))
	// hashed := hash.Sum(nil)
	// signed, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// r.Header.Set("Signature", fmt.Sprintf(`keyId="%s",algorithm="rsa-sha256",headers="(request-target) digest date",signature="%s"`, pubKeyId, base64.StdEncoding.EncodeToString(signed)))

	prefs := []httpsig.Algorithm{httpsig.RSA_SHA256}
	// The "Date" and "Digest" headers must already be set on r, as well as r.URL.
	headersToSign := []string{httpsig.RequestTarget, "date", "digest"}
	signer, _, err := httpsig.NewSigner(prefs, httpsig.DigestSha256, headersToSign, httpsig.Signature, 60)
	if err != nil {
		log.Fatal(err)
	}
	// If r were a http.ResponseWriter, call SignResponse instead.
	if err := signer.SignRequest(privateKey, pubKeyId, r, body); err != nil {
		log.Fatal(err)
	}
}

func dumpRequest(r *http.Request) {
	fmt.Printf("%s %s %s\n", r.Method, r.URL, r.Proto)
	for k := range r.Header {
		fmt.Printf("%s: %s\n", k, r.Header.Get(k))
	}
	fmt.Println()
}

func dumpResponse(r *http.Response) {
	fmt.Printf("%s %s\n", r.Proto, r.Status)
	for k := range r.Header {
		fmt.Printf("%s: %s\n", k, r.Header.Get(k))
	}
	fmt.Println()
	io.Copy(os.Stdout, r.Body)
	fmt.Println()
}
