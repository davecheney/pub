package mastodon

import (
	"fmt"
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
)

func ConversationsIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var statuses []*models.Status
	scope := env.DB.Scopes(models.PaginateConversation(r)).Where("visibility = ?", "direct")
	switch r.URL.Query().Get("local") {
	case "":
		scope = scope.Joins("Actor")
	default:
		scope = scope.Joins("Actor").Where("Actor.domain = ?", r.Host)
	}

	if err := scope.Order("statuses.id desc").Find(&statuses).Error; err != nil {
		return err
	}

	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/public?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/public?min_id=%d>; rel=\"prev\"", r.Host, statuses[len(statuses)-1].ID, r.Host, statuses[0].ID))
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(statuses, serialise.Status))
}
