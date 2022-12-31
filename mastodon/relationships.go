package mastodon

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/davecheney/m/internal/activitypub"
	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/internal/snowflake"
	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
)

type Relationships struct {
	service *Service
}

func (r *Relationships) Show(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	targets := req.URL.Query()["id"]
	targets = append(targets, req.URL.Query()["id[]"]...)
	var resp []any
	for _, target := range targets {
		id, err := strconv.ParseUint(target, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tid := snowflake.ID(id)
		var rel models.Relationship
		if err := r.service.DB().Preload("Target").FirstOrCreate(&rel, models.Relationship{ActorID: user.Actor.ID, TargetID: tid}).Error; err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		resp = append(resp, serializeRelationship(&rel))
	}
	toJSON(w, resp)
}

func (r *Relationships) Create(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target models.Actor
	if err := r.service.DB().First(&target, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	err = r.sendFollowRequest(user, &target)
	if err != nil {
		fmt.Println("sendFollowRequest failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := m.NewService(r.service.DB())
	rel, err := svc.Relationships().Follow(user.Actor, &target)
	toJSON(w, serializeRelationship(rel))
}

func (r *Relationships) sendFollowRequest(account *models.Account, target *models.Actor) error {
	client, err := activitypub.NewClient(account.Actor.PublicKeyID(), account.PrivateKey)
	if err != nil {
		return err
	}
	return client.Follow(account.Actor.URI, target.URI)
}

func (r *Relationships) Destroy(w http.ResponseWriter, req *http.Request) {
	user, err := r.service.authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target models.Actor
	if err := r.service.DB().First(&target, chi.URLParam(req, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	err = r.sendUnfollowRequest(user, &target)
	if err != nil {
		fmt.Println("sendUnfollowRequest failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := m.NewService(r.service.DB())
	rel, err := svc.Relationships().Unfollow(user.Actor, &target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeRelationship(rel))
}

func (r *Relationships) sendUnfollowRequest(account *models.Account, target *models.Actor) error {
	client, err := activitypub.NewClient(account.Actor.PublicKeyID(), account.PrivateKey)
	if err != nil {
		return err
	}
	return client.Unfollow(account.Actor.URI, target.URI)
}

type Relationship struct {
	ID                  snowflake.ID `json:"id,string"`
	Following           bool         `json:"following"`
	ShowingReblogs      bool         `json:"showing_reblogs"`
	Notifying           bool         `json:"notifying"`
	FollowedBy          bool         `json:"followed_by"`
	Blocking            bool         `json:"blocking"`
	BlockedBy           bool         `json:"blocked_by"`
	Muting              bool         `json:"muting"`
	MutingNotifications bool         `json:"muting_notifications"`
	Requested           bool         `json:"requested"`
	DomainBlocking      bool         `json:"domain_blocking"`
	Endorsed            bool         `json:"endorsed"`
	Note                string       `json:"note"`
}

func serializeRelationship(rel *models.Relationship) *Relationship {
	return &Relationship{
		ID:                  rel.TargetID,
		Following:           rel.Following,
		ShowingReblogs:      true,  // todo
		Notifying:           false, // todo
		FollowedBy:          rel.FollowedBy,
		Blocking:            rel.Blocking,
		BlockedBy:           rel.BlockedBy,
		Muting:              rel.Muting,
		MutingNotifications: false,
		Requested:           false,
		DomainBlocking:      false,
		Endorsed:            false,
		Note: func() string {
			// FirstOrCreate won't preload the Target
			// so it will be zero. :(
			if rel.Target != nil {
				return rel.Target.Note
			} else {
				return ""
			}
		}(),
	}
}
