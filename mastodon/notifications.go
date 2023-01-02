package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/to"
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
	to.JSON(w, []map[string]any{})
}
