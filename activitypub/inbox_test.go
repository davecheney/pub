package activitypub

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublishedAndUpdated(t *testing.T) {
	t.Run("published and updated are the same when updated is missing ", func(t *testing.T) {
		require := require.New(t)

		obj := map[string]any{
			"published": time.Now().Format(time.RFC3339),
		}
		published, updated, err := publishedAndUpdated(obj)
		require.NoError(err)
		require.Equal(published, updated)
	})
	t.Run("published and updated are the same when updated is empty", func(t *testing.T) {
		require := require.New(t)

		obj := map[string]any{
			"published": time.Now().Format(time.RFC3339),
			"updated":   "",
		}
		published, updated, err := publishedAndUpdated(obj)
		require.NoError(err)
		require.Equal(published, updated)
	})
	t.Run("published and updated are the same when updated is invalid", func(t *testing.T) {
		require := require.New(t)

		obj := map[string]any{
			"published": time.Now().Format(time.RFC3339),
			"updated":   "invalid",
		}
		published, updated, err := publishedAndUpdated(obj)
		require.NoError(err)
		require.Equal(published, updated)
	})
	t.Run("updated is newer than published when updated is valid", func(t *testing.T) {
		require := require.New(t)

		published := time.Now().Add(-1 * time.Hour)
		updated := time.Now()
		obj := map[string]any{
			"published": published.Format(time.RFC3339),
			"updated":   updated.Format(time.RFC3339),
		}
		published, updated, err := publishedAndUpdated(obj)
		require.NoError(err)
		require.Equal(published, published)
	})
}
