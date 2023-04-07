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
