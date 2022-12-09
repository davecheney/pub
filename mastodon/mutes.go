package mastodon

import "net/http"

type Mutes struct {
	service *Service
}

func (m *Mutes) Index(w http.ResponseWriter, r *http.Request) {
	_, err := m.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	toJSON(w, []any{})
}
