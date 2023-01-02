// package to contains functions for converting between types.
package to

import (
	"net/http"

	"github.com/go-json-experiment/json"
)

// JSON writes the given object to the response body as JSON.
// If obj is a nil slice, an empty JSON array is written.
// If obj is a nil map, an empty JSON object is written.
// If obj is a nil pointer, a null is written.
func JSON(w http.ResponseWriter, obj any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.MarshalOptions{}.MarshalFull(json.EncodeOptions{
		Indent: "  ",
	}, w, obj)
}
