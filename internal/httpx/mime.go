package httpx

import (
	"net/http"
	"strings"
)

// MediaType returns the media type of the request.
func MediaType(req *http.Request) string {
	typ := strings.Split(req.Header.Get("Content-Type"), ";")[0]
	if typ == "" {
		typ = "application/octet-stream"
	}
	return typ
}
