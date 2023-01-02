package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/to"
)

type Emojis struct {
	service *Service
}

func (e *Emojis) Index(w http.ResponseWriter, r *http.Request) {
	to.JSON(w, []any{})
}
