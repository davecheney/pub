package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/to"
)

func NotificationsIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
	return to.JSON(w, []map[string]any{})
}
