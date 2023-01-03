package mastodon

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/go-chi/chi/v5"
)

func AccountsShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var actor models.Actor
	if err := env.DB.First(&actor, chi.URLParam(r, "id")).Error; err != nil {
		return httpx.Error(http.StatusNotFound, err)
	}
	return to.JSON(w, serialiseAccount(&actor))
}

func AccountsVerifyCredentials(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	return to.JSON(w, serialiseCredentialAccount(user))
}

func AccountsStatusesShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	tx := env.DB.Preload("Actor").Where("actor_id = ?", chi.URLParam(r, "id"))

	// todo: use pagination
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 40 {
		limit = 20
	}
	tx = tx.Limit(limit)
	sinceID, _ := strconv.Atoi(r.URL.Query().Get("since_id"))
	if sinceID > 0 {
		tx = tx.Where("id > ?", sinceID)
	}

	var statuses []*models.Status
	if err := tx.Order("id desc").Find(&statuses).Error; err != nil {
		return err
	}

	return to.JSON(w, algorithms.Map(statuses, serialiseStatus))
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
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/accounts/%s/followers?max_id=%d>; rel=\"next\", <https://%s/api/v1/accounts/%s/followers?min_id=%d>; rel=\"prev\"", r.Host, chi.URLParam(r, "id"), followers[len(followers)-1].TargetID, r.Host, chi.URLParam(r, "id"), followers[0].TargetID))
	}
	return to.JSON(w, algorithms.Map(algorithms.Map(followers, relationshipTarget), serialiseAccount))
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
		// TODO don't send if we're at the end of the list
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/accounts/%s/following?max_id=%d>; rel=\"next\", <https://%s/api/v1/accounts/%s/following?min_id=%d>; rel=\"prev\"", r.Host, chi.URLParam(r, "id"), following[len(following)-1].TargetID, r.Host, chi.URLParam(r, "id"), following[0].TargetID))
	}
	return to.JSON(w, algorithms.Map(algorithms.Map(following, relationshipTarget), serialiseAccount))
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
	return to.JSON(w, serialiseAccount(account.Actor))
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

	return to.JSON(w, algorithms.Map(lists, serialiseList))
}
