// package media is a read through cache for media files.
package media

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"net/http"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/go-chi/chi/v5"
)

func Show(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	kind := chi.URLParam(r, "kind")
	switch kind {
	case "avatar":
		return showAvatar(env, w, r)
	case "header":
		return showHeader(env, w, r)
	case "original":
		return showOriginal(env, w, r)
	default:
		return httpx.Error(http.StatusNotFound, fmt.Errorf("unknown kind %q", kind))
	}
}

func showAvatar(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var actor models.Actor
	if err := env.DB.Take(&actor, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	return fetch(w, stringOrDefault(actor.Avatar, "https://avatars.githubusercontent.com/u/1024?v=4"))
}

func showHeader(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var actor models.Actor
	if err := env.DB.Take(&actor, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	return fetch(w, stringOrDefault(actor.Header, "https://static.ma-cdn.net/headers/original/missing.png"))
}

func showOriginal(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var att models.StatusAttachment
	if err := env.DB.Take(&att, chi.URLParam(r, "id")).Error; err != nil {
		fmt.Println(err)
		return httpx.Error(http.StatusNotFound, err)
	}
	return fetch(w, att.URL)
}

func fetch(w http.ResponseWriter, url string) error {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return httpx.Error(http.StatusBadGateway, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return httpx.Error(http.StatusBadGateway, fmt.Errorf("unexpected status code %d", resp.StatusCode))
	}

	// read the first 512 bytes to determine the content type.
	buf := bufio.NewReader(resp.Body)
	head, err := buf.Peek(512)
	if err != nil && err != io.EOF {
		return httpx.Error(http.StatusBadGateway, err)
	}
	contentType := http.DetectContentType(head)
	w.Header().Set("Content-Type", contentType)
	_, err = io.Copy(w, buf)
	return err
}

func ProxyAvatarURL(actor *models.Actor) string {
	url := stringOrDefault(actor.Avatar, "https://avatars.githubusercontent.com/u/1024?v=4")
	return fmt.Sprintf("https://cheney.net/media/avatar/%s/%d", b64Hash(sha256.New(), url), actor.ID)
}

func ProxyHeaderURL(actor *models.Actor) string {
	url := stringOrDefault(actor.Header, "https://avatars.githubusercontent.com/u/1024?v=4")
	return fmt.Sprintf("https://cheney.net/media/header/%s/%d", b64Hash(sha256.New(), url), actor.ID)
}

func b64Hash(h hash.Hash, s string) string {
	h.Reset()
	io.WriteString(h, s)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func stringOrDefault(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}
