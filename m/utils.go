package m

import (
	"net/http"
	"strconv"
	"strings"
)

func utoa(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}

// mediaType returns the media type of the request.
func mediaType(req *http.Request) string {
	typ := strings.Split(req.Header.Get("Content-Type"), ";")[0]
	if typ == "" {
		typ = "application/octet-stream"
	}
	return typ
}
