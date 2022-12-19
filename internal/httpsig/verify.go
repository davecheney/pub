package httpsig

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Verify verifies signature of the request.
func Verify(req *http.Request, keyFn func(keyID string) (crypto.PublicKey, error)) error {
	sigHeader := req.Header.Get("Signature")
	if sigHeader == "" {
		return errors.New("Signature header is missing")
	}

	var (
		pubKey  crypto.PublicKey
		algo    string
		sig     []byte
		headers []string
		err     error
	)
	for _, part := range strings.Split(sigHeader, ",") {
		k, v := strings.SplitN(part, "=", 2)[0], strings.SplitN(part, "=", 2)[1]
		switch k {
		case "keyId":
			keyID := strings.Trim(v, "\"")
			pubKey, err = keyFn(keyID)
			if err != nil {
				return err
			}
		case "algorithm":
			algo = strings.Trim(v, "\"")
		case "headers":
			headers = strings.Split(strings.Trim(v, "\""), " ")
		case "signature":
			sig, err = base64.StdEncoding.DecodeString(strings.Trim(v, "\""))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown signature part: %s", part)
		}
	}

	var sb strings.Builder
	for _, header := range headers {
		switch header {
		case RequestTarget:
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
	io.WriteString(hash, strings.TrimRight(sb.String(), "\n")) // remove trailing newline
	digest := hash.Sum(nil)

	switch algo {
	case "rsa-sha256":
		return rsaVerify(pubKey, digest, sig)
	default:
		return fmt.Errorf("unknown algorithm: %s", algo)
	}
}

func rsaVerify(pubKey crypto.PublicKey, digest, sig []byte) error {
	return rsa.VerifyPKCS1v15(pubKey.(*rsa.PublicKey), crypto.SHA256, digest, sig)
}
