package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
)

type Contexts struct {
	service *Service
}

func (c *Contexts) Show(w http.ResponseWriter, r *http.Request) {
	user, err := c.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var status models.Status
	if err := c.service.db.First(&status, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// load conversation statuses
	var statuses []models.Status
	query := c.service.db.Joins("Actor").Preload("Reblog").Preload("Reblog.Actor").Preload("Attachments").Preload("Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.Where("conversation_id = ?", status.ConversationID).Find(&statuses).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ancestors, decendants := thread(status.ID, statuses)
	to.JSON(w, struct {
		Ancestors   []map[string]any `json:"ancestors"`
		Descendants []map[string]any `json:"descendants"`
	}{
		Ancestors: func() []map[string]interface{} {
			a := make([]map[string]interface{}, 0) // make sure we return an empty array, not null
			for _, s := range ancestors {
				a = append(a, serialiseStatus(s))
			}
			return a
		}(),
		Descendants: func() []map[string]interface{} {
			a := make([]map[string]interface{}, 0) // make sure we return an empty array, not null
			for _, s := range decendants {
				a = append(a, serialiseStatus(s))
			}
			return a
		}(),
	})
}

// thread sorts statuses into a tree, it returns the statuses
// preceding id, and statuses following id.
func thread(id snowflake.ID, statuses []models.Status) ([]*models.Status, []*models.Status) {
	type link struct {
		parent   *link
		status   *models.Status
		children []*link
	}
	ids := make(map[snowflake.ID]*link)
	for i := range statuses {
		ids[statuses[i].ID] = &link{status: &statuses[i]}
	}

	for _, l := range ids {
		if l.status.InReplyToID != nil {
			parent, ok := ids[*l.status.InReplyToID]
			if ok {
				// watch out for deleted toots
				l.parent = parent
				parent.children = append(parent.children, l)
			}
		}
	}

	var ancestors []*models.Status
	var l = ids[id].parent
	for l != nil {
		ancestors = append(ancestors, l.status)
		l = l.parent
	}
	reverse(ancestors)

	var descendants []*models.Status
	var walk func(*link)
	walk = func(l *link) {
		for _, c := range l.children {
			descendants = append(descendants, c.status)
			walk(c)
		}
	}
	walk(ids[id])
	return ancestors, descendants
}

func reverse[T any](a []T) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}
