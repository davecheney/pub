// Package httpx is a convenience wrapper around the http.ServeMux type that
// allows us to return errors from our handlers.
// see https://blog.questionable.services/article/http-handler-error-handling-revisited/ for more details.
package httpx

import (
	"errors"
	"net/http"

	"github.com/go-json-experiment/json"
	"golang.org/x/exp/slog"
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
	Log() *slog.Logger
}

// HandlerFunc adapts a function that returns an error to an http.HandlerFunc.
func HandlerFunc[E env](envFn func(r *http.Request) E, fn func(E, http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		env := envFn(r)
		err := fn(env, w, r)
		if err != nil {
			status := http.StatusInternalServerError
			if se := new(StatusError); errors.As(err, &se) {
				status = se.Code
			}
			env.Log().Error("pub/http", "method", r.Method, "path", r.URL.Path, "status", status, "error", err)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			json.MarshalWrite(w, map[string]any{
				"error": err.Error(),
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
