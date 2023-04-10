// Package httpx is a convenience wrapper around the http.ServeMux type that
// allows us to return errors from our handlers.
// see https://blog.questionable.services/article/http-handler-error-handling-revisited/ for more details.
package httpx

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-json-experiment/json"
)

// Error is a convenience function for returning an error with an associated HTTP status code.
func Error(code int, err error) error {
	return &StatusError{code, err}
}

// StatusError represents an error with an associated HTTP status code.
type StatusError struct {
	Code int
	Err  error
}

// Allows StatusError to satisfy the error interface.
func (se *StatusError) Error() string {
	return se.Err.Error()
}

// Returns our HTTP status code.
func (se *StatusError) Status() int {
	return se.Code
}

// HandlerFunc adapts a function that returns an error to an http.HandlerFunc.
func HandlerFunc[E any](envFn func(r *http.Request) *E, fn func(*E, http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		env := envFn(r)
		err := fn(env, w, r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			if se := new(StatusError); errors.As(err, &se) {
				log.Printf("HTTP: method: %s, path: %s, status: %d, error: %s", r.Method, r.URL.Path, se.Status(), err)
				w.WriteHeader(se.Status())
				json.MarshalFull(w, map[string]any{
					"error": se.Error(),
				})
				return
			}
			log.Printf("HTTP: method: %s, path: %s, status: %d, error: %s", r.Method, r.URL.Path, http.StatusInternalServerError, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.MarshalFull(w, map[string]any{
				"error": http.StatusInternalServerError,
			})
		}
	}
}

// Redirect returns a 302 redirect to the specified URI.
func Redirect(w http.ResponseWriter, uri string) error {
	w.Header().Set("Location", uri)
	w.WriteHeader(302)
	return nil
}
