package mastodon

import (
	"net/http"

	"github.com/davecheney/m/m"
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

	var lists []m.AccountList
	if err := l.service.DB().Model(user).Association("Lists").Find(&lists); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []any
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

	var resp []any

	toJSON(w, resp)
}
