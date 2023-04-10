package to_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/davecheney/pub/internal/to"
	"github.com/stretchr/testify/require"
)

// mockResponseWriter is an io.Writer that satisfies the http.ResponseWriter
// interface.
type mockResponseWriter struct {
	bytes.Buffer
}

func (w *mockResponseWriter) Header() http.Header {
	return http.Header{}
}

func (w *mockResponseWriter) WriteHeader(int) {}

func TestToJSONReturnsEmptyArrayForNilSlice(t *testing.T) {
	require := require.New(t)

	var s []string = nil
	var out mockResponseWriter
	err := to.JSON(&out, s)
	require.NoError(err)
	require.Equal("[]", out.String())
}

func TestToJSONReturnsEmptyObjectForNilMap(t *testing.T) {
	require := require.New(t)

	var m map[string]string = nil
	var out mockResponseWriter
	err := to.JSON(&out, m)
	require.NoError(err)
	require.Equal("{}", out.String())
}

func TestToJSONReturnsAnEmptyArrayForKeyWithNilSlice(t *testing.T) {
	require := require.New(t)

	m := map[string]interface{}{
		"foo": []string(nil),
	}
	var out mockResponseWriter
	err := to.JSON(&out, m)
	require.NoError(err)
	require.Equal("{\n  \"foo\": []\n}", out.String())
}

func TestTOJSONDoesNotEscapeHTML(t *testing.T) {
	require := require.New(t)

	m := map[string]interface{}{
		"foo": "<p>Hello, world!</p>",
	}
	var out mockResponseWriter
	err := to.JSON(&out, m)
	require.NoError(err)
	require.Equal("{\n  \"foo\": \"<p>Hello, world!</p>\"\n}", out.String())
}
