package mastodon

import (
	"net/http"

	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Mutes struct {
	service *Service
}

func (svc *Mutes) Index(w http.ResponseWriter, r *http.Request) {
	user, err := svc.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var mutes []m.Relationship
	if err := svc.service.DB().Joins("Target").Find(&mutes, "actor_id = ? and muting = true", user.Actor.ID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	var resp []any
	for _, a := range mutes {
		resp = append(resp, serialize(a.Target))
	}
	toJSON(w, resp)
}

func (svc *Mutes) Create(w http.ResponseWriter, r *http.Request) {
	user, err := svc.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target m.Actor
	if err := svc.service.DB().First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var rel m.Relationship
	if err := svc.service.DB().Joins("Target").First(&rel, "actor_id = ? and target_id = ?", user.Actor.ID, target.ID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rel = m.Relationship{
			ActorID:  user.Actor.ID,
			TargetID: target.ID,
			Target:   &target,
		}
	}

	rel.Muting = true
	if err := svc.service.DB().Clauses(clause.OnConflict{UpdateAll: true}).Create(&rel).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeRelationship(&rel))
}

func (svc *Mutes) Destroy(w http.ResponseWriter, r *http.Request) {
	user, err := svc.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target m.Actor
	if err := svc.service.DB().First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var rel m.Relationship
	if err := svc.service.DB().Joins("Target").First(&rel, "actor_id = ? and target_id = ?", user.Actor.ID, target.ID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rel.Muting = false
	if err := svc.service.DB().Clauses(clause.OnConflict{UpdateAll: true}).Create(&rel).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	toJSON(w, serializeRelationship(&rel))
}
