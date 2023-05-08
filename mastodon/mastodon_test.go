package mastodon

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davecheney/pub/internal/snowflake"
	"github.com/stretchr/testify/require"
)

func TestLinkHeader(t *testing.T) {
	require := require.New(t)

	req, err := http.NewRequest("POST", "https://example.com/api/v1/timelines/public", nil)
	require.NoError(err)

	rec := httptest.NewRecorder()
	oldest := snowflake.ID(110330528023225442)
	newest := oldest + 1000

	linkHeader(rec, req, newest, oldest)

	require.Equal(rec.Header()["Link"], []string{
		`<https://example.com/api/v1/timelines/public?max_id=110330528023225442>; rel="next", <https://example.com/api/v1/timelines/public?min_id=110330528023226442>; rel="prev"`,
	})
}
