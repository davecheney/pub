package m

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-json-experiment/json"
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

// toJSON writes the given object to the response body as JSON.
func toJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return json.MarshalFull(w, obj)
}
