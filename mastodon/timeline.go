package mastodon

import (
	"fmt"
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
)

func TimelinesHome(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var followingIDs []int64
	if err := env.DB.Model(&models.Relationship{ActorID: user.Actor.ID}).Where("following = true").Pluck("target_id", &followingIDs).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}
	followingIDs = append(followingIDs, int64(user.ID))

	var statuses []*models.Status
	scope := env.DB.Scopes(models.PaginateStatuses(r)).Where("(actor_id IN (?) AND in_reply_to_actor_id is null) or (actor_id in (?) and in_reply_to_actor_id IN (?))", followingIDs, followingIDs, followingIDs)
	scope = scope.Joins("Actor").Preload("Reblog").Preload("Reblog.Actor").Preload("Attachments").Preload("Reaction", "actor_id = ?", user.Actor.ID)
	if err := scope.Find(&statuses).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/home?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/home?min_id=%d>; rel=\"prev\"", r.Host, statuses[len(statuses)-1].ID, r.Host, statuses[0].ID))
	}
	return to.JSON(w, algorithms.Map(statuses, serialiseStatus))
}

func TimelinesPublic(env *Env, w http.ResponseWriter, r *http.Request) error {
	var statuses []*models.Status
	scope := env.DB.Scopes(models.PaginateStatuses(r)).Where("visibility = ? and reblog_id is null and in_reply_to_id is null", "public")
	switch r.URL.Query().Get("local") {
	case "true":
		scope = scope.Joins("Actor").Where("Actor.domain = ?", r.Host)
	default:
		scope = scope.Joins("Actor")
	}
	scope = scope.Preload("Attachments")
	if err := scope.Find(&statuses).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/public?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/public?min_id=%d>; rel=\"prev\"", r.Host, statuses[len(statuses)-1].ID, r.Host, statuses[0].ID))
	}
	return to.JSON(w, algorithms.Map(statuses, serialiseStatus))
}

func TimelinesListShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var listMembers []int64
	if err := env.DB.Model(&models.AccountListMember{}).Where("account_list_id = ? ", chi.URLParam(r, "id")).Pluck("member_id", &listMembers).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	var statuses []*models.Status
	scope := env.DB.Scopes(models.PaginateStatuses(r)).Where("(actor_id IN (?) AND in_reply_to_actor_id is null) or (actor_id in (?) and in_reply_to_actor_id IN (?))", listMembers, listMembers, listMembers)
	scope = scope.Joins("Actor").Preload("Reblog").Preload("Reblog.Actor").Preload("Attachments").Preload("Reaction", "actor_id = ?", user.Actor.ID)
	if err := scope.Find(&statuses).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	// if len(statuses) > 0 {
	// 	w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/home?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/home?min_id=%d>; rel=\"prev\"", r.Host, statuses[len(statuses)-1].ID, r.Host, statuses[0].ID))
	// }
	return to.JSON(w, algorithms.Map(statuses, serialiseStatus))
}
