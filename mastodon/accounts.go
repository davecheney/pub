package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
)

func AccountsShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var actor models.Actor
	if err := env.DB.Scopes(models.PreloadActor).Take(&actor, "id = ? ", chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Account(&actor))
}

func AccountsVerifyCredentials(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.CredentialAccount(user))
}

func AccountsStatusesShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var statuses []*models.Status
	query := env.DB.Scopes(
		models.PaginateStatuses(r),
		models.PreloadStatus,
		models.MaybeExcludeReplies(r),
		models.MaybeExcludeReblogs(r),
		models.MaybePinned(r),
	)
	query = query.Preload("Actor")
	query = query.Preload("Reaction", &models.Reaction{ActorID: user.Actor.ID}) // reactions
	query = query.Preload("Reblog.Reaction", &models.Reaction{ActorID: user.Actor.ID})
	if err := query.Find(&statuses, "statuses.actor_id = ?", chi.URLParam(r, "id")).Error; err != nil {
		return err
	}

	sortStatuses(statuses) // PaginateStatuses doesn't sort, so we have to do it ourselves.

	if len(statuses) > 0 {
		linkHeader(w, r, statuses[0].ID, statuses[len(statuses)-1].ID)
	}
	seralise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(statuses, seralise.Status))
}

func AccountsFollowersShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var followers []*models.Relationship
	if err := env.DB.Scopes(models.PaginateRelationship(r)).Preload("Target").Where("actor_id = ? and followed_by = true", chi.URLParam(r, "id")).Find(&followers).Error; err != nil {
		return err
	}

	if len(followers) > 0 {
		linkHeader(w, r, followers[0].TargetID, followers[len(followers)-1].TargetID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(algorithms.Map(followers, relationshipTarget), serialise.Account))
}

func relationshipTarget(rel *models.Relationship) *models.Actor {
	return rel.Target
}

func AccountsFollowingShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var following []*models.Relationship
	if err := env.DB.Scopes(models.PaginateRelationship(r)).Preload("Target").Where("actor_id = ? and following = true", chi.URLParam(r, "id")).Find(&following).Error; err != nil {
		return err
	}

	if len(following) > 0 {
		linkHeader(w, r, following[0].TargetID, following[len(following)-1].TargetID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(algorithms.Map(following, relationshipTarget), serialise.Account))
}

func AccountsUpdateCredentials(env *Env, w http.ResponseWriter, r *http.Request) error {
	account, err := env.authenticate(r)
	if err != nil {
		return err
	}

	if err := r.ParseForm(); err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	if r.Form.Get("display_name") != "" {
		account.Actor.DisplayName = r.Form.Get("display_name")
	}
	if r.Form.Get("note") != "" {
		account.Actor.Note = r.Form.Get("note")
	}

	if err := env.DB.Save(account).Error; err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.Account(account.Actor))
}

func AccountsShowListMembership(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	memberID, err := snowflake.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	accountLists := env.DB.Select("account_list_id").Where(&models.AccountListMember{MemberID: memberID}).Table("account_list_members")
	var lists []*models.AccountList
	if err := env.DB.Where("id IN (?)", accountLists).Find(&lists).Error; err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(lists, serialise.List))
}

func AccountsFamiliarFollowersShow(env *Env, w http.ResponseWriter, req *http.Request) error {
	_, err := env.authenticate(req)
	if err != nil {
		return err
	}
	ids := req.URL.Query()["id"]
	ids = append(ids, req.URL.Query()["id[]"]...)
	// serialise := Serialiser{req: req}

	type ff struct {
		ID       snowflake.ID `json:"id"`
		Accounts []any        `json:"accounts"`
	}

	var resp []any
	for _, i := range ids {
		id, err := snowflake.Parse(i)
		if err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
		resp = append(resp, &ff{
			ID: id,
		})
	}
	return to.JSON(w, resp)
}

func AccountsFeaturedTagsShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
	return to.JSON(w, []any{})
}
