package mastodon

import (
	"errors"
	"net/http"

	"github.com/davecheney/pub/internal/httpx"
)

func PushSubscriptionCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
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
	if err := httpx.Params(r, &body); err != nil {
		return err
	}
	return httpx.Error(http.StatusForbidden, errors.New("this action is outside the authorized scopes"))
}
