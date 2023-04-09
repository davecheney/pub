// package media is a read through cache for media files.
package media

import (
	"bufio"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"

	"io"
	"net/http"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
	"github.com/nfnt/resize"
)

func Avatar(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var actor models.Actor
	if err := env.DB.Take(&actor, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	if actor.Avatar == "" {
		return httpx.Error(http.StatusNotFound, fmt.Errorf("no avatar for actor %q", actor.ID))
	}
	return stream(w, actor.Avatar)
}

func Header(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var actor models.Actor
	if err := env.DB.Take(&actor, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	if actor.Header == "" {
		return httpx.Error(http.StatusNotFound, fmt.Errorf("no header for actor %q", actor.ID))
	}
	return stream(w, actor.Header)
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

// Preview returns a preview of the attachment in the format requested by the
// file extension in the URL.
func Preview(env *models.Env, w http.ResponseWriter, r *http.Request) error {
	var att models.StatusAttachment

	if err := env.DB.Take(&att, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	resp, err := http.DefaultClient.Get(fmt.Sprintf("https://%s/media/original/%d.%s", r.Host, att.ID, att.Extension()))
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
	switch ext := chi.URLParam(r, "ext"); ext {
	case "jpg":
		w.Header().Set("Content-Type", "image/jpeg")
		return jpeg.Encode(w, img, nil)
	// case "png":
	// 	w.Header().Set("Content-Type", "image/png")
	// 	return png.Encode(w, img)
	case "gif":
		w.Header().Set("Content-Type", "image/gif")
		return gif.Encode(w, img, nil)
	default:
		return httpx.Error(http.StatusNotAcceptable, fmt.Errorf("unknown extension %q", ext))
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
