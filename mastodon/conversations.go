package mastodon

import (
	"fmt"
	"net/http"

	"github.com/davecheney/m/internal/models"
)

type Conversations struct {
	service *Service
}

func (c *Conversations) Index(w http.ResponseWriter, r *http.Request) {
	_, err := c.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var statuses []models.Status
	scope := c.service.db.Scopes(models.PaginateConversation(r)).Where("visibility = ?", "direct")
	switch r.URL.Query().Get("local") {
	case "":
		scope = scope.Joins("Actor")
	default:
		scope = scope.Joins("Actor").Where("Actor.domain = ?", r.Host)
	}

	if err := scope.Order("statuses.id desc").Find(&statuses).Error; err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := []any{} // ensure we return an array
	for _, status := range statuses {
		resp = append(resp, serialiseStatus(&status))
	}
	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/public?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/public?min_id=%d>; rel=\"prev\"", r.Host, statuses[len(statuses)-1].ID, r.Host, statuses[0].ID))
	}
	toJSON(w, resp)
}
