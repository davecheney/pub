package mastodon

import "net/http"

type Blocks struct {
	service *Service
}

func (b *Blocks) Index(w http.ResponseWriter, r *http.Request) {
	_, err := b.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	toJSON(w, []any{})
}
