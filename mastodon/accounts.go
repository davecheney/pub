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
	query = query.Preload("Reaction", &models.Reaction{ActorID: user.Actor.ObjectID}) // reactions
	query = query.Preload("Reblog.Reaction", &models.Reaction{ActorID: user.Actor.ObjectID})
	if err := query.Where("statuses.actor_id = ?", chi.URLParam(r, "id")).Find(&statuses).Error; err != nil {
		return err
	}

	sortStatuses(statuses) // PaginateStatuses doesn't sort, so we have to do it ourselves.

	if len(statuses) > 0 {
		linkHeader(w, r, statuses[0].ObjectID, statuses[len(statuses)-1].ObjectID)
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
	if err := env.DB.Scopes(models.PaginateRelationship(r)).Preload("Target").Preload("Target.Attributes").Where("actor_id = ? and following = true", chi.URLParam(r, "id")).Find(&following).Error; err != nil {
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

	// if r.Form.Get("display_name") != "" {
	// 	account.Actor.DisplayName = r.Form.Get("display_name")
	// }
	// if r.Form.Get("note") != "" {
	// 	account.Actor.Note = r.Form.Get("note")
	// }

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
	user, err := env.authenticate(req)
	if err != nil {
		return err
	}
	var params struct {
		ID    snowflake.ID   `schema:"id"`
		IDs   []snowflake.ID `schema:"id[]"`
		Limit int            `schema:"limit"` // ignored
	}
	if err := httpx.Params(req, &params); err != nil {
		return err
	}
	var resp []FamiliarFollowers
	serialiser := Serialiser{req: req}
	for _, id := range append(params.IDs, params.ID) {
		if id == 0 {
			continue
		}
		followers := env.DB.Select("target_id").Where("actor_id = ? and following = true", id).Table("relationships")
		var commonFollowers []*models.Relationship
		if err := env.DB.Preload("Target").Preload("Target.Attributes").Where("actor_id = ? and following = true and target_id in (?)", user.Actor.ObjectID, followers).Find(&commonFollowers).Error; err != nil {
			return httpx.Error(http.StatusInternalServerError, err)
		}
		resp = append(resp, FamiliarFollowers{
			ID:       id,
			Accounts: algorithms.Map(algorithms.Map(commonFollowers, relationshipTarget), serialiser.Account),
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
