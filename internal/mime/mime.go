// package mime contains helper functions for setting the Content-Type header.
package mime

import (
	"net/http"
	"strings"
)

// mediaType returns the media type of the request.
func MediaType(req *http.Request) string {
	typ := strings.Split(req.Header.Get("Content-Type"), ";")[0]
	if typ == "" {
		typ = "application/octet-stream"
	}
	return typ
}
