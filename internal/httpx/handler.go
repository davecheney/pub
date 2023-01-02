package httpx

// see https://blog.questionable.services/article/http-handler-error-handling-revisited/

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

type env interface {
	any
}

// HandlerFunc adapts a function that returns an error to an http.HandlerFunc.
func HandlerFunc[E any](envFn func(r *http.Request) *E, fn func(*E, http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(envFn(r), w, r)
		if err != nil {
			if se := new(StatusError); errors.As(err, &se) {
				log.Printf("HTTP: path: %s, status: %d, error: %s", r.URL.Path, se.Status(), se.Error())
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(se.Status())
				json.MarshalFull(w, map[string]any{
					"error": se.Error(),
				})
				return
			}
			log.Printf("HTTP: path: %s, status: %d, error: %s", r.URL.Path, http.StatusInternalServerError, err)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
