package mastodon

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

func StatusesCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	actor := user.Actor
	var toot struct {
		Status      string        `json:"status"`
		InReplyToID *snowflake.ID `json:"in_reply_to_id,string"`
		Sensitive   bool          `json:"sensitive"`
		SpoilerText string        `json:"spoiler_text"`
		Visibility  string        `json:"visibility"`
		Language    string        `json:"language"`
		ScheduledAt *time.Time    `json:"scheduled_at,omitempty"`
	}
	if err := json.UnmarshalFull(r.Body, &toot); err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	var conv *models.Conversation
	if toot.InReplyToID != nil {
		var parent models.Status
		if err := env.DB.Take(&parent, *toot.InReplyToID).Error; err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
		conv, err = models.NewConversations(env.DB).FindOrCreate(parent.ConversationID, toot.Visibility)
		if err != nil {
			return err
		}
	} else {
		conv, err = models.NewConversations(env.DB).New(toot.Visibility)
		if err != nil {
			return err
		}
	}

	createdAt := time.Now()
	id := snowflake.TimeToID(createdAt)
	status := models.Status{
		ID:             id,
		UpdatedAt:      createdAt,
		ActorID:        actor.ID,
		Actor:          actor,
		ConversationID: conv.ID,
		InReplyToID:    toot.InReplyToID,
		URI:            fmt.Sprintf("https://%s/users/%s/%d", actor.Domain, actor.Name, id),
		Sensitive:      toot.Sensitive,
		SpoilerText:    toot.SpoilerText,
		Visibility:     toot.Visibility,
		Language:       toot.Language,
		Note:           toot.Status,
	}
	if err := env.DB.Create(&status).Error; err != nil {
		return err
	}
	return to.JSON(w, serialiseStatus(&status))
}

func StatusesDestroy(env *Env, w http.ResponseWriter, r *http.Request) error {
	account, err := env.authenticate(r)
	if err != nil {
		return err
	}
	actor := account.Actor
	var status models.Status
	if err := env.DB.Joins("Actor").Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	if status.ActorID != actor.ID {
		return httpx.Error(http.StatusForbidden, errors.New("forbidden"))
	}
	if err := env.DB.Delete(&status).Error; err != nil {
		return err
	}
	return to.JSON(w, serialiseStatus(&status))
}

func StatusesShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var status models.Status
	query := env.DB.Joins("Actor").Scopes(models.PreloadStatus)
	query = query.Preload("Reaction", "actor_id = ?", user.Actor.ID) // reactions
	query = query.Preload("Reblog.Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	return to.JSON(w, serialiseStatus(&status))
}

func StatusesHistoryShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var status models.Status
	query := env.DB.Joins("Actor").Scopes(models.PreloadStatus)
	query = query.Preload("Reaction", "actor_id = ?", user.Actor.ID) // reactions
	query = query.Preload("Reblog.Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	return to.JSON(w, []any{serialiseStatusEdit(&status)})
}

func StatusesContextsShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var status models.Status
	query := env.DB.Joins("Actor").Scopes(models.PreloadStatus)
	query = query.Preload("Reaction", "actor_id = ?", user.Actor.ID) // reactions
	query = query.Preload("Reblog.Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}

	// load conversation statuses
	var statuses []models.Status
	query = env.DB.Joins("Actor").Scopes(models.PreloadStatus)
	query = query.Preload("Reaction", "actor_id = ?", user.Actor.ID) // reactions
	query = query.Preload("Reblog.Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.Where(&models.Status{ConversationID: status.ConversationID}).Find(&statuses).Error; err != nil {
		return err
	}

	if len(statuses) > 0 {
		w.Header().Add("Link", fmt.Sprintf(`<https://%s/%s?min_id=%d>; rel="prev"`, r.Host, r.URL, statuses[len(statuses)-1].ID))
	}
	ancestors, descendants := thread(status.ID, statuses)
	return to.JSON(w, struct {
		Ancestors   []*Status `json:"ancestors"`
		Descendants []*Status `json:"descendants"`
	}{
		Ancestors:   algorithms.Map(ancestors, serialiseStatus),
		Descendants: algorithms.Map(descendants, serialiseStatus),
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
