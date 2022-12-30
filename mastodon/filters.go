package mastodon

import (
	"net/http"
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

	toJSON(w, []map[string]any{})
}
