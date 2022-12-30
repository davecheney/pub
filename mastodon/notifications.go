package mastodon

import (
	"net/http"
)

type Notifications struct {
	service *Service
}

func (n *Notifications) Index(w http.ResponseWriter, r *http.Request) {
	_, err := n.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	toJSON(w, []map[string]any{})
}
