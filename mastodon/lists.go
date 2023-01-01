package mastodon

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/internal/snowflake"
	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
)

type Lists struct {
	service *Service
}

func (l *Lists) Index(w http.ResponseWriter, r *http.Request) {
	user, err := l.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var lists []models.AccountList
	if err := l.service.db.Model(user).Association("Lists").Find(&lists); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := []any{} // ensure we return an array, not null
	for _, list := range lists {
		resp = append(resp, serialiseList(&list))
	}
	toJSON(w, resp)
}

func (l *Lists) Show(w http.ResponseWriter, r *http.Request) {
	user, err := l.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var lists []models.AccountList
	if err := l.service.db.Model(&models.AccountList{}).Joins("Members").Where("member_id = ?", chi.URLParam(r, "id")).Find(&lists, "account_id = ?", user.ID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := []any{} // ensure we return an array, not null
	for _, list := range lists {
		resp = append(resp, serialiseList(&list))
	}
	toJSON(w, resp)
}

func (l *Lists) Create(w http.ResponseWriter, r *http.Request) {
	user, err := l.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	fmt.Println("params", params)
	list := models.AccountList{
		ID:            snowflake.Now(),
		Title:         params.Title,
		RepliesPolicy: params.RepliesPolicy,
	}
	if err := l.service.db.Model(user).Association("Lists").Append(&list); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	toJSON(w, serialiseList(&list))
}

func (l *Lists) AddMembers(w http.ResponseWriter, r *http.Request) {
	// x, _ := httputil.DumpRequest(r, true)
	// fmt.Println(string(x))
	_, err := l.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	listID, err := snowflake.Parse(chi.URLParam(r, "id"))
	if err != nil {
		fmt.Println("error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var params struct {
		AccountIDs []string `json:"account_ids"`
	}
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/x-www-form-urlencoded":
		params.AccountIDs = strings.Split(r.FormValue("account_ids[]"), ",")
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	fmt.Println("params", params)

	list := models.AccountList{
		ID: listID,
	}
	for _, id := range params.AccountIDs {
		memberID, err := snowflake.Parse(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := l.service.db.Model(&list).Association("Members").Append(&models.AccountListMember{
			AccountListID: listID,
			MemberID:      memberID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	toJSON(w, map[string]any{})
}

func (l *Lists) RemoveMembers(w http.ResponseWriter, r *http.Request) {
	// x, _ := httputil.DumpRequest(r, true)
	// fmt.Println(string(x))
	_, err := l.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	listID, err := snowflake.Parse(chi.URLParam(r, "id"))
	if err != nil {
		fmt.Println("error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var params struct {
		AccountIDs []string `json:"account_ids"`
	}
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/x-www-form-urlencoded":
		params.AccountIDs = strings.Split(r.FormValue("account_ids[]"), ",")
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	list := models.AccountList{
		ID: listID,
	}
	for _, id := range params.AccountIDs {
		memberID, err := snowflake.Parse(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := l.service.db.Model(&list).Association("Members").Delete(&models.AccountListMember{
			AccountListID: listID,
			MemberID:      memberID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	toJSON(w, map[string]any{})
}

func (l *Lists) ViewMembers(w http.ResponseWriter, r *http.Request) {
	_, err := l.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var members []models.AccountListMember
	if err := l.service.db.Model(&models.AccountList{}).Joins("Members").Preload("Member").Where("id = ?", chi.URLParam(r, "id")).Find(&members).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := []any{} // ensure we return an array, not null
	for _, member := range members {
		resp = append(resp, serialiseAccount(member.Member))
	}
	toJSON(w, resp)
}
