package mastodon

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/models"
	"github.com/stretchr/testify/require"
)

func TestSerialiserMediaOriginalURL(t *testing.T) {

	t.Run("Attachment with supported media type returns cached URL", func(t *testing.T) {
		require := require.New(t)

		att := &models.Attachment{
			ID:        snowflake.Now(),
			URL:       "https://example.com/u/user.jpg",
			MediaType: "image/jpeg",
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		require.Equal(fmt.Sprintf("https://%s/media/original/%d.jpg", req.Host, att.ID), s.mediaOriginalURL(att))
	})

	t.Run("Attachment with unsupported media type returns original URL", func(t *testing.T) {
		require := require.New(t)

		att := &models.Attachment{
			ID:        snowflake.Now(),
			URL:       "https://example.com/u/user.mp4",
			MediaType: "video/mp4",
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		require.Equal(att.URL, s.mediaOriginalURL(att))
	})

}

func TestSerialiserMediaPreviewURL(t *testing.T) {

	t.Run("Attachment with zero height returns empty string", func(t *testing.T) {
		require := require.New(t)

		att := &models.Attachment{
			Height: 0,
			Width:  100,
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		require.Empty(s.mediaPreviewURL(att))
	})

	t.Run("Attachment with zero width returns empty string", func(t *testing.T) {
		require := require.New(t)

		att := &models.Attachment{
			Height: 100,
			Width:  0,
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		require.Empty(s.mediaPreviewURL(att))
	})

	t.Run("Attachment with zero height and width returns empty string", func(t *testing.T) {
		require := require.New(t)

		att := &models.Attachment{
			Height: 0,
			Width:  0,
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		require.Empty(s.mediaPreviewURL(att))
	})

	t.Run("Attachment with height and width below preview max returns empty string", func(t *testing.T) {
		require := require.New(t)

		att := &models.Attachment{
			Height: PREVIEW_MAX_HEIGHT - 100,
			Width:  PREVIEW_MAX_WIDTH - 100,
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		require.Empty(s.mediaPreviewURL(att))
	})

	t.Run("Attachment with height and width above preview max returns preview URL", func(t *testing.T) {
		require := require.New(t)

		id := snowflake.Now()

		att := &models.Attachment{
			ID:        id,
			Height:    PREVIEW_MAX_HEIGHT + 100,
			Width:     PREVIEW_MAX_WIDTH + 100,
			MediaType: "image/jpeg",
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		want := fmt.Sprintf("https://example.com/media/preview/%d.jpg", id)
		require.Equal(want, s.mediaPreviewURL(att))
	})

	t.Run("Attachment with height and width below preview max returns original URL", func(t *testing.T) {
		require := require.New(t)

		id := snowflake.Now()

		att := &models.Attachment{
			ID:        id,
			Height:    PREVIEW_MAX_HEIGHT - 100,
			Width:     PREVIEW_MAX_WIDTH - 100,
			MediaType: "image/jpeg",
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		want := fmt.Sprintf("https://example.com/media/original/%d.jpg", id)
		require.Equal(want, s.mediaPreviewURL(att))
	})

	t.Run("Attachment with unknown media type returns empty string", func(t *testing.T) {
		require := require.New(t)

		att := &models.Attachment{
			Height:    PREVIEW_MAX_HEIGHT + 100,
			Width:     PREVIEW_MAX_WIDTH + 100,
			MediaType: "image/unknown",
		}
		req, err := http.NewRequest("GET", "https://example.com/u/user", nil)
		require.NoError(err)

		s := Serialiser{req}

		require.Empty(s.mediaPreviewURL(att))
	})
}

func TestFocus(t *testing.T) {
	t.Run("MediaAttachment with no FocalPoint returns nil", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Height: 100,
			Width:  100,
		}
		require.Nil(focus(att))
	})
	t.Run("MediaAttachment with FocalPoint returns MetaFocus", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Height: 100,
			Width:  100,
			FocalPoint: models.FocalPoint{
				X: 0.5,
				Y: 0.5,
			},
		}
		require.Equal(&MetaFocus{
			X: 0.5,
			Y: 0.5,
		}, focus(att))
	})
	t.Run("MediaAttachment with incomplete FocalPoint returns incomplete MetaFocus", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Height: 100,
			Width:  100,
			FocalPoint: models.FocalPoint{
				X: 0.5,
			},
		}
		require.Equal(&MetaFocus{
			X: 0.5,
		}, focus(att))
	})
}

func TestOriginalMetaFormat(t *testing.T) {
	t.Run("MediaAttachment with no dimensions returns nil", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{}
		require.Nil(originalMetaFormat(att))
	})
	t.Run("MediaAttachment with incomplete dimensions returns nil", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Height: 100,
		}
		require.Nil(originalMetaFormat(att))
	})
	t.Run("MediaAttachment with dimensions returns MetaFormat", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Width:  640,
			Height: 480,
		}
		require.Equal(&MetaFormat{
			Width:  640,
			Height: 480,
			Size:   "640x480",
			Aspect: 640.0 / 480.0,
		}, originalMetaFormat(att))
	})
}

func TestSmallMetaFormat(t *testing.T) {
	t.Run("MediaAttachment with no dimensions returns nil", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{}
		require.Nil(smallMetaFormat(att))
	})
	t.Run("MediaAttachment with incomplete dimensions returns nil", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Height:    100,
			MediaType: "image/jpeg",
		}
		require.Nil(smallMetaFormat(att))
	})
	t.Run("JPEG MediaAttachment with dimension less that preview size returns original MetaFormat", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Width:     PREVIEW_MAX_WIDTH - 100,
			Height:    PREVIEW_MAX_HEIGHT - 100,
			MediaType: "image/jpeg",
		}
		require.Equal(&MetaFormat{
			Width:  att.Width,
			Height: att.Height,
			Size:   fmt.Sprintf("%dx%d", att.Width, att.Height),
			Aspect: float64(att.Width) / float64(att.Height),
		}, smallMetaFormat(att))
	})
	t.Run("JPEG MediaAttachment with one dimension less that PREVIEW_MAX_WIDTH returns scaled MetaFormat", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Width:     PREVIEW_MAX_WIDTH / 2,
			Height:    PREVIEW_MAX_HEIGHT,
			MediaType: "image/jpeg",
		}
		width := att.Width * PREVIEW_MAX_HEIGHT / att.Height
		height := PREVIEW_MAX_HEIGHT

		require.Equal(&MetaFormat{
			Width:  width,
			Height: height,
			Size:   fmt.Sprintf("%dx%d", width, height),
			Aspect: float64(width) / float64(height),
		}, smallMetaFormat(att))
	})
	t.Run("MediaAttachment with one dimension less that PREVIEW_MAX_HEIGHT returns scaled MetaFormat", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Width:     PREVIEW_MAX_WIDTH,
			Height:    PREVIEW_MAX_HEIGHT / 2,
			MediaType: "image/jpeg",
		}
		height := att.Height * PREVIEW_MAX_WIDTH / att.Width
		width := PREVIEW_MAX_WIDTH

		require.Equal(&MetaFormat{
			Width:  width,
			Height: height,
			Size:   fmt.Sprintf("%dx%d", width, height),
			Aspect: float64(width) / float64(height),
		}, smallMetaFormat(att))
	})
	t.Run("Wide JPEG MediaAttachment with dimensions returns scaled MetaFormat", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Width:     640,
			Height:    480,
			MediaType: "image/jpeg",
		}
		height := att.Height * PREVIEW_MAX_WIDTH / att.Width
		width := PREVIEW_MAX_WIDTH

		require.Equal(&MetaFormat{
			Width:  width,
			Height: height,
			Size:   fmt.Sprintf("%dx%d", width, height),
			Aspect: float64(width) / float64(height),
		}, smallMetaFormat(att))
	})
	t.Run("Tall JPEG MediaAttachment with dimensions returns scaled MetaFormat", func(t *testing.T) {
		require := require.New(t)
		att := &models.Attachment{
			Width:     480,
			Height:    640,
			MediaType: "image/jpeg",
		}
		width := float64(att.Width) * PREVIEW_MAX_HEIGHT / float64(att.Height)
		height := PREVIEW_MAX_HEIGHT

		require.Equal(&MetaFormat{
			Width:  int(width),
			Height: height,
			Size:   fmt.Sprintf("%dx%d", int(width), height),
			Aspect: width / float64(height),
		}, smallMetaFormat(att))
	})
}
