package m

import (
	"net/http"
	"strconv"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

// withTX runs each function in the given slice within the supplied transaction.
func withTX(tx *gorm.DB, fns ...func(tx *gorm.DB) error) error {
	for _, fn := range fns {
		if err := fn(tx); err != nil {
			return err
		}
	}
	return nil
}

func utoa(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}

// toJSON writes the given object to the response body as JSON.
func toJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return json.MarshalFull(w, obj)
}

func ptr[T any](v T) *T {
	return &v
}
