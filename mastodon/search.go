package mastodon

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/internal/webfinger"
	"github.com/go-json-experiment/json"
)

type Search struct {
	service *Service
}

func (s *Search) Index(w http.ResponseWriter, r *http.Request) {
	_, err := s.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	q := r.URL.Query().Get("q")
	typ := r.URL.Query().Get("type")
	if strings.Contains(q, "@") {
		typ = "accounts"
	}
	switch typ {
	case "accounts":
		s.searchAccounts(w, r, q)
	// case "hashtags":
	// 	s.searchHashtags(w, r, q)
	default:
		s.searchStatuses(w, r, q)
	}
}

func (s *Search) searchAccounts(w http.ResponseWriter, r *http.Request, q string) {
	var actor *models.Actor
	var err error
	switch r.URL.Query().Get("resolve") == "true" {
	case true:
		// true to fix up search query
		switch {
		case strings.HasPrefix(q, "https://"):
			u, err := url.Parse(q)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			user := strings.TrimPrefix(u.Path[1:], "@")
			q = "acct:" + user + "@" + u.Host
			fallthrough
		case strings.Contains(q, "@"):
			fmt.Println("webfinger", q)
			acct, err := webfinger.Parse(q)
			if err != nil {
				fmt.Println("webfinger.Parse", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			wf, err := acct.Fetch(r.Context())
			if err != nil {
				fmt.Println("acct.Fetch", acct, wf, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			q, err = wf.ActivityPub()
			if err != nil {
				fmt.Println("wf.ActivityPub", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		fetcher := s.service.Service.Actors().NewRemoteActorFetcher()
		actor, err = s.service.Service.Actors().FindOrCreate(q, fetcher.Fetch)
	default:
		actor, err = s.service.Service.Actors().FindByURI(q)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp = map[string]any{
		"accounts": []any{
			serialiseAccount(actor),
		},
		"hashtags": []any{},
		"statuses": []any{},
	}
	toJSON(w, resp)
}

func (s *Search) searchStatuses(w http.ResponseWriter, r *http.Request, q string) {
	var status *models.Status
	var err error
	switch r.URL.Query().Get("resolve") == "true" {
	case true:
		fetcher := s.service.Service.Statuses().NewRemoteStatusFetcher()
		status, err = s.service.Service.Statuses().FindOrCreate(q, fetcher.Fetch)
	default:
		status, err = s.service.Service.Statuses().FindByURI(q)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp = map[string]any{
		"accounts": []any{},
		"hashtags": []any{},
		"statuses": []any{
			serialiseStatus(status),
		},
	}
	toJSON(w, resp)
}

func marshalIndent(v any) ([]byte, error) {
	b, err := json.MarshalOptions{}.Marshal(json.EncodeOptions{
		Indent: "\t", // indent for readability
	}, v)
	return b, err
}
