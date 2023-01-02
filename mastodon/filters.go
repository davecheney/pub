package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/to"
)

type Filters struct {
	service *Service
}

func (f *Filters) Index(w http.ResponseWriter, r *http.Request) {
	_, err := f.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	to.JSON(w, []map[string]any{})
}
