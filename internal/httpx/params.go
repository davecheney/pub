package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/gorilla/schema"
)

// Params decodes the request parameters of a POST request into the given struct
// based on the Content-Type header. It returns an error if the Content-Type is
// not supported.
func Params(r *http.Request, v interface{}) error {
	switch r.Method {
	case "GET", "HEAD":
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			return Error(http.StatusBadRequest, err)
		}
		if err := schema.NewDecoder().Decode(v, values); err != nil {
			return Error(http.StatusBadRequest, err)
		}
	case "POST":
		switch mediaType(r) {
		case "application/json":
			if err := json.UnmarshalFull(r.Body, v); err != nil {
				return Error(http.StatusBadRequest, err)
			}
		case "":
			// ice cubes, why you gotta do me like this?
			values, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil {
				return Error(http.StatusBadRequest, err)
			}
			if err := schema.NewDecoder().Decode(v, values); err != nil {
				return Error(http.StatusBadRequest, err)
			}
		case "application/x-www-form-urlencoded":
			if err := r.ParseForm(); err != nil {
				return err
			}
			if err := schema.NewDecoder().Decode(v, r.Form); err != nil {
				return Error(http.StatusBadRequest, err)
			}
		case "multipart/form-data":
			if err := r.ParseMultipartForm(0); err != nil {
				return err
			}
			if err := schema.NewDecoder().Decode(v, r.PostForm); err != nil {
				return Error(http.StatusBadRequest, err)
			}
		default:
			return Error(http.StatusUnsupportedMediaType, fmt.Errorf("unsupported media type: %q", r.Header.Get("Content-Type")))
		}
	default:
		return Error(http.StatusMethodNotAllowed, errors.New("unsupported method: "+r.Method))
	}
	return nil
}

// mediaType returns the media type of the request.
func mediaType(req *http.Request) string {
	return strings.Split(req.Header.Get("Content-Type"), ";")[0]
}
