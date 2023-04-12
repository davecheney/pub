package mastodon

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func StatusesCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var toot struct {
		Status      string        `json:"status" schema:"status"`
		InReplyToID *snowflake.ID `json:"in_reply_to_id,string" schema:"in_reply_to_id"`
		Sensitive   bool          `json:"sensitive" schema:"sensitive"`
		SpoilerText string        `json:"spoiler_text" schema:"spoiler_text"`
		Visibility  string        `json:"visibility" schema:"visibility"`
		Language    string        `json:"language" schema:"language"`
		ScheduledAt *time.Time    `json:"scheduled_at,omitempty" schema:"scheduled_at"`
	}
	if err := httpx.Params(r, &toot); err != nil {
		return err
	}

	var parent models.Status
	var conv *models.Conversation
	if toot.InReplyToID != nil {
		if err := env.DB.Preload("Conversation").Take(&parent, *toot.InReplyToID).Error; err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
		conv = parent.Conversation
	} else {
		conv = &models.Conversation{
			Visibility: models.Visibility(toot.Visibility),
		}
	}

	createdAt := time.Now()
	id := snowflake.TimeToID(createdAt)
	status := models.Status{
		ID:           id,
		UpdatedAt:    createdAt,
		Actor:        user.Actor,
		Conversation: conv,
		InReplyToID: func() *snowflake.ID {
			if parent.ID != 0 {
				return &parent.ID
			} else {
				return nil
			}
		}(),
		InReplyToActorID: func() *snowflake.ID {
			if parent.ActorID != 0 {
				return &parent.ActorID
			} else {
				return nil
			}
		}(),
		URI:         fmt.Sprintf("https://%s/users/%s/%d", user.Actor.Domain, user.Actor.Name, id),
		Sensitive:   toot.Sensitive,
		SpoilerText: toot.SpoilerText,
		Visibility:  models.Visibility(toot.Visibility),
		Language:    toot.Language,
		Note:        toot.Status,
	}
	if err := env.DB.Create(&status).Error; err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(&status))
}

func StatusesReblogCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var status models.Status
	if err := env.DB.Joins("Actor").Scopes(models.PreloadStatus).Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}

	reblog, err := models.NewReactions(env.DB).Reblog(&status, user.Actor)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(reblog))
}

func StatusesReblogDestroy(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var status models.Status
	if err := env.DB.Joins("Actor").Scopes(models.PreloadStatus).Take(&status, chi.URLParam(r, "id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}

	unblogged, err := models.NewReactions(env.DB).Unreblog(&status, user.Actor)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(unblogged))
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
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(&status))
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
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Status(&status))
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
	serialise := Serialiser{req: r}
	return to.JSON(w, []any{serialise.StatusEdit(&status)})
}

func StatusesFavouritesShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var favouriters []*models.Actor
	query := env.DB.Joins("JOIN reactions ON reactions.actor_id = actors.id and reactions.status_id = ? and reactions.favourited = ?", chi.URLParam(r, "id"), true)
	query = query.Scopes(models.PreloadActor, models.PaginateActors(r))
	if err := query.Order("id desc").Find(&favouriters).Error; err != nil {
		return err
	}

	if len(favouriters) > 0 {
		linkHeader(w, r, favouriters[0].ID, favouriters[len(favouriters)-1].ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(favouriters, serialise.Account))
}

func StatusesReblogsShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var rebloggers []*models.Actor
	query := env.DB.Joins("JOIN statuses ON statuses.actor_id = actors.id and statuses.reblog_id = ?", chi.URLParam(r, "id"))
	query = query.Scopes(models.PreloadActor, models.PaginateActors(r))
	if err := query.Order("id desc").Find(&rebloggers).Error; err != nil {
		return err
	}

	if len(rebloggers) > 0 {
		linkHeader(w, r, rebloggers[0].ID, rebloggers[len(rebloggers)-1].ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(rebloggers, serialise.Account))
}

func StatusesContextsShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var status models.Status
	query := env.DB.Joins("Actor") // don't need to preload everything, just the actor to prove it exists
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

	ancestors, descendants := thread(status.ID, statuses)
	serialise := Serialiser{req: r}
	return to.JSON(w, struct {
		Ancestors   []*Status `json:"ancestors"`
		Descendants []*Status `json:"descendants"`
	}{
		Ancestors:   algorithms.Map(ancestors, serialise.Status),
		Descendants: algorithms.Map(descendants, serialise.Status),
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
	algorithms.Reverse(ancestors)

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
