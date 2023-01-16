// package to contains functions for converting between types.
package to

import (
	"io"
	"net/http"

	"github.com/go-json-experiment/json"
)

// JSON writes the given object to the response body as JSON.
// If obj is a nil slice, an empty JSON array is written.
// If obj is a nil map, an empty JSON object is written.
// If obj is a nil pointer, a null is written.
func JSON(rw http.ResponseWriter, obj any, writerOpts ...func(w io.Writer) io.Writer) error {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	w := io.Writer(rw)
	for _, opt := range writerOpts {
		w = opt(w)
	}
	return json.MarshalOptions{}.MarshalFull(json.EncodeOptions{
		Indent: "  ",
	}, w, obj)
}
