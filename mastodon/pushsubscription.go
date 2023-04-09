package mastodon

import (
	"errors"
	"net/http"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/mime"
	"github.com/go-json-experiment/json"
)

func PushSubscriptionCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
	switch mime.MediaType(r) {
	case "application/json":
		var body struct {
			Data struct {
				Policy string `json:"policy"`
				Alerts struct {
					Follow    bool `json:"follow"`
					Favourite bool `json:"favourite"`
					Reblog    bool `json:"reblog"`
					Mention   bool `json:"mention"`
				} `json:"alerts"`
			} `json:"data"`
			Subscription struct {
				Endpoint string `json:"endpoint"`
				Keys     struct {
					P256DH string `json:"p256dh"`
					Auth   string `json:"auth"`
				} `json:"keys"`
			} `json:"subscription"`
		}
		if err := json.UnmarshalFull(r.Body, &body); err != nil {
			return err
		}
		return httpx.Error(http.StatusForbidden, errors.New("This action is outside the authorized scopes"))
	default:
		return httpx.Error(http.StatusUnsupportedMediaType, errors.New("Unsupported media type"))
	}
}
