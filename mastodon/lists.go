package mastodon

import (
	"net/http"

	"github.com/davecheney/m/internal/models"
)

type Lists struct {
	service *Service
}

func (l *Lists) Index(w http.ResponseWriter, r *http.Request) {
	user, err := l.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var lists []models.AccountList
	if err := l.service.db.Model(user).Association("Lists").Find(&lists); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := []any{} // ensure we return an array, not null
	for _, list := range lists {
		resp = append(resp, map[string]any{
			"id":             toString(list.ID),
			"title":          list.Title,
			"replies_policy": list.RepliesPolicy,
		})
	}
	toJSON(w, resp)
}

func (l *Lists) Show(w http.ResponseWriter, r *http.Request) {
	_, err := l.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	resp := []any{} // ensure we return an array, not null

	toJSON(w, resp)
}
