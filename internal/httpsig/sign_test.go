package httpsig

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"

	"github.com/go-fed/httpsig"
	"github.com/stretchr/testify/require"
)

func TestSignRequest(t *testing.T) {
	require := require.New(t)
	req, err := http.NewRequest("GET", "https://example.com/users/foo", nil)
	req.Header.Set("Accept", "application/ld+json")
	require.NoError(err)

	const keyID = "https://example.com#main-key"
	privatekey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(err)
	pubKey := &privatekey.PublicKey

	err = Sign(req, keyID, privatekey, nil)
	require.NoError(err)

	verifier, err := httpsig.NewVerifier(req)
	require.NoError(err)
	require.Equal(keyID, verifier.KeyId())
	err = verifier.Verify(pubKey, httpsig.RSA_SHA256)
	require.NoError(err, "req.Signature: %s", req.Header.Get("Signature"))
}
