package mastodon

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
)

func ListsIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var lists []*models.AccountList
	if err := env.DB.Model(user).Association("Lists").Find(&lists); err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(lists, serialise.List))
}

func ListsShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var list models.AccountList
	if err := env.DB.Model(user).Association("Lists").Find(&list, chi.URLParam(r, "id")); err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.List(&list))
}

func ListsCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var params struct {
		Title         string `json:"title"`
		RepliesPolicy string `json:"replies_policy"`
	}
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/x-www-form-urlencoded":
		params.Title = r.FormValue("title")
		params.RepliesPolicy = r.FormValue("replies_policy")
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
	default:
		return httpx.Error(http.StatusUnsupportedMediaType, fmt.Errorf("unsupported media type"))
	}
	fmt.Println("params", params)
	list := models.AccountList{
		ID:            snowflake.Now(),
		Title:         params.Title,
		RepliesPolicy: params.RepliesPolicy,
	}
	if err := env.DB.Model(user).Association("Lists").Append(&list); err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, serialise.List(&list))
}

func ListsAddMembers(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	listID, err := snowflake.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	var params struct {
		AccountIDs []string `json:"account_ids"`
	}
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/x-www-form-urlencoded":
		params.AccountIDs = strings.Split(r.FormValue("account_ids[]"), ",")
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
	default:
		return httpx.Error(http.StatusUnsupportedMediaType, fmt.Errorf("unsupported media type"))
	}
	fmt.Println("params", params)

	list := models.AccountList{
		ID: listID,
	}
	for _, id := range params.AccountIDs {
		memberID, err := snowflake.Parse(id)
		if err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
		if err := env.DB.Model(&list).Association("Members").Append(&models.AccountListMember{
			AccountListID: listID,
			MemberID:      memberID,
		}); err != nil {
			return err
		}
	}

	return to.JSON(w, map[string]any{})
}

func ListsRemoveMembers(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	listID, err := snowflake.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	var params struct {
		AccountIDs []string `json:"account_ids"`
	}
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/x-www-form-urlencoded":
		params.AccountIDs = strings.Split(r.FormValue("account_ids[]"), ",")
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
	default:
		return httpx.Error(http.StatusUnsupportedMediaType, fmt.Errorf("unsupported media type"))
	}

	list := models.AccountList{
		ID: listID,
	}
	for _, id := range params.AccountIDs {
		memberID, err := snowflake.Parse(id)
		if err != nil {
			return httpx.Error(http.StatusBadRequest, err)
		}
		if err := env.DB.Model(&list).Association("Members").Delete(&models.AccountListMember{
			AccountListID: listID,
			MemberID:      memberID,
		}); err != nil {
			return err
		}
	}

	return to.JSON(w, map[string]any{})
}

func ListsViewMembers(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var members []*models.AccountListMember
	if err := env.DB.Joins("Member").Find(&members, "account_list_id", chi.URLParam(r, "id")).Error; err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(algorithms.Map(members, listMember), serialise.Account))
}

func listMember(list *models.AccountListMember) *models.Actor {
	return list.Member
}
