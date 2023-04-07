package mastodon

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/internal/webfinger"
	"github.com/davecheney/pub/models"
)

func SearchIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query().Get("q")
	typ := r.URL.Query().Get("type")
	if strings.Contains(q, "@") {
		typ = "accounts"
	}
	switch typ {
	case "accounts":
		return searchAccounts(env, w, r, q)
	// case "hashtags":
	// 	s.searchHashtags(w, r, q)
	default:
		return searchStatuses(env, w, r, q)
	}
}

func searchAccounts(env *Env, w http.ResponseWriter, r *http.Request, q string) error {
	var actor *models.Actor
	var err error
	switch r.URL.Query().Get("resolve") == "true" {
	case true:
		// true to fix up search query
		switch {
		case strings.HasPrefix(q, "https://"):
			u, err := url.Parse(q)
			if err != nil {
				return httpx.Error(http.StatusBadRequest, err)
			}
			user := strings.TrimPrefix(u.Path[1:], "@")
			q = "acct:" + user + "@" + u.Host
			fallthrough
		case strings.Contains(q, "@"):
			fmt.Println("webfinger", q)
			acct, err := webfinger.Parse(q)
			if err != nil {
				fmt.Println("webfinger.Parse", err)
				return httpx.Error(http.StatusBadRequest, err)
			}
			wf, err := acct.Fetch(r.Context())
			if err != nil {
				fmt.Println("acct.Fetch", acct, wf, err)
				return httpx.Error(http.StatusBadRequest, err)
			}
			q, err = wf.ActivityPub()
			if err != nil {
				fmt.Println("wf.ActivityPub", err)
				return httpx.Error(http.StatusBadRequest, err)
			}
		}
		// find admin of this request's domain
		var instance models.Instance
		if err := env.DB.Joins("Admin").Preload("Admin.Actor").Where("domain = ?", r.Host).First(&instance).Error; err != nil {
			return httpx.Error(http.StatusInternalServerError, err)
		}
		fetcher := activitypub.NewRemoteActorFetcher(instance.Admin, env.DB)
		actor, err = models.NewActors(env.DB).FindOrCreate(q, fetcher.Fetch)
	default:
		actor, err = models.NewActors(env.DB).FindByURI(q)
	}
	if err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	serialise := Serialiser{req: r}
	var resp = map[string]any{
		"accounts": []any{
			serialise.Account(actor),
		},
		"hashtags": []any{},
		"statuses": []any{},
	}
	return to.JSON(w, resp)
}

func searchStatuses(env *Env, w http.ResponseWriter, r *http.Request, q string) error {
	var status *models.Status
	var err error
	switch r.URL.Query().Get("resolve") == "true" {
	case true:
		// find admin of this request's domain
		var instance models.Instance
		if err := env.DB.Joins("Admin").Preload("Admin.Actor").Where("domain = ?", r.Host).First(&instance).Error; err != nil {
			return httpx.Error(http.StatusInternalServerError, err)
		}
		fetcher := activitypub.NewRemoteStatusFetcher(instance.Admin, env.DB)
		status, err = models.NewStatuses(env.DB).FindOrCreate(q, fetcher.Fetch)
	default:
		status, err = models.NewStatuses(env.DB).FindByURI(q)
	}
	if err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}
	serialise := Serialiser{req: r}
	var resp = map[string]any{
		"accounts": []any{},
		"hashtags": []any{},
		"statuses": []any{
			serialise.Status(status),
		},
	}
	return to.JSON(w, resp)
}
