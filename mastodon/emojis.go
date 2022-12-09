package mastodon

import (
	"net/http"
)

type Emojis struct {
	service *Service
}

func (e *Emojis) Index(w http.ResponseWriter, r *http.Request) {
	toJSON(w, []any{})
}
