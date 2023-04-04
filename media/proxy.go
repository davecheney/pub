// package media is a read through cache for media files.
package media

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/nfnt/resize"
)

func Avatar(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var actor models.Actor
	if err := env.DB.Take(&actor, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	return stream(w, stringOrDefault(actor.Avatar, "https://avatars.githubusercontent.com/u/1024?v=4"))
}

func Header(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var actor models.Actor
	if err := env.DB.Take(&actor, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	return stream(w, stringOrDefault(actor.Header, "https://static.ma-cdn.net/headers/original/missing.png"))
}

func Original(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var att models.StatusAttachment
	if err := env.DB.Take(&att, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	return stream(w, att.URL)
}

const (
	PREVIEW_MAX_WIDTH  = 560
	PREVIEW_MAX_HEIGHT = 415
)

func Preview(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var att models.StatusAttachment

	if err := env.DB.Take(&att, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	ext := chi.URLParam(r, "ext")
	resp, err := http.DefaultClient.Get(fmt.Sprintf("https://%s/media/original/%d.%s", r.Host, att.ID, ext))
	if err != nil {
		return httpx.Error(http.StatusBadGateway, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return httpx.Error(http.StatusBadGateway, fmt.Errorf("unexpected status code %d", resp.StatusCode))
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return httpx.Error(http.StatusBadGateway, err)
	}

	b := img.Bounds()
	if b.Dx() > PREVIEW_MAX_WIDTH || b.Dy() > PREVIEW_MAX_HEIGHT {
		img = resize.Thumbnail(PREVIEW_MAX_WIDTH, PREVIEW_MAX_HEIGHT, img, resize.Lanczos3)
	}
	switch ext {
	case "jpg":
		w.Header().Set("Content-Type", "image/jpeg")
		return jpeg.Encode(w, img, nil)
	case "png":
		w.Header().Set("Content-Type", "image/png")
		return png.Encode(w, img)
	case "gif":
		w.Header().Set("Content-Type", "image/gif")
		return gif.Encode(w, img, nil)
	default:
		return httpx.Error(http.StatusNotFound, fmt.Errorf("unknown extension %q", ext))
	}
}

// stream streams the content of the url to the http.ResponseWriter.
func stream(w http.ResponseWriter, url string) error {
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
