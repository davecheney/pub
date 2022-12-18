// Package httpsig implements the HTTP Signature scheme as defined in draft-cavage-http-signatures-10.
package httpsig

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-fed/httpsig"
)

const (
	// RequestTarget is the pseudo-header used to sign the request target.
	RequestTarget = "(request-target)"
)

// Sign signs the request using the given keyID and privateKey.
func Sign(req *http.Request, keyID string, privateKey crypto.PrivateKey, body []byte) error {
	req.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // Date must be in GMT, not UTC 🤯
	headersToSign := []string{
		RequestTarget,
	}
	switch req.Method {
	case "GET":
		headersToSign = append(headersToSign, "host", "date", "accept")
	case "POST":
		headersToSign = append(headersToSign, "date", "digest")
		addDigest(req, body)
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
		case "Digest", "digest":
			sb.WriteString("digest: ")
			sb.WriteString(req.Header.Get("Digest"))
		default:
			return fmt.Errorf("unknown header to sign: %s", header)
		}
		sb.WriteString("\n")
	}
	hash := sha256.New()
	hash.Write(bytes.TrimRight(sb.Bytes(), "\n")) // remove trailing newline
	digest := hash.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey.(*rsa.PrivateKey), crypto.SHA256, digest)
	if err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString(sig)
	req.Header.Set("Signature", fmt.Sprintf(`keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`, keyID, strings.Join(headersToSign, " "), enc))
	return nil
}

func addDigest(req *http.Request, body []byte) {
	hash := sha256.New()
	hash.Write(body)
	digest := hash.Sum(nil)
	req.Header.Set("Digest", fmt.Sprintf("SHA-256=%s", base64.StdEncoding.EncodeToString(digest)))
}
