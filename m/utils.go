package m

import (
	"net/http"
	"strconv"

	"github.com/go-json-experiment/json"
)

func utoa(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}

// toJSON writes the given object to the response body as JSON.
func toJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return json.MarshalFull(w, obj)
}
