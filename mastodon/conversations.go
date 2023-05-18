package mastodon

import (
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
	query := env.DB.Scopes(models.PaginateConversation(r), models.PreloadStatus).Where("visibility = ?", "direct")
	switch r.URL.Query().Get("local") {
	case "":
		// nothing
	default:
		query = query.Where("Actor.domain = ?", r.Host)
	}
	if err := query.Order("statuses.object_id desc").Find(&statuses).Error; err != nil {
		return err
	}
	if len(statuses) > 0 {
		linkHeader(w, r, statuses[0].ObjectID, statuses[len(statuses)-1].ObjectID)
	}

	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(statuses, serialise.Status))
}
