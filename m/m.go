package m

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
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

func intFromAny(v any) int {
	switch v := v.(type) {
	case int:
		return v
	case float64:
		// shakes fist at json number type
		return int(v)
	}
	return 0
}

func splitAcct(acct string) (string, string, error) {
	url, err := url.Parse(acct)
	if err != nil {
		return "", "", fmt.Errorf("splitAcct: %w", err)
	}
	return path.Base(url.Path), url.Host, nil
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}
