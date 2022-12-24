package mastodon

import (
	"net/http"

	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Blocks struct {
	service *Service
}

func (b *Blocks) Index(w http.ResponseWriter, r *http.Request) {
	user, err := b.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var blocks []m.Relationship
	if err := b.service.DB().Joins("Target").Find(&blocks, "actor_id = ? and blocking = true", user.Actor.ID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	var resp []any
	for _, a := range blocks {
		resp = append(resp, serialize(a.Target))
	}
	toJSON(w, resp)
}

func (b *Blocks) Create(w http.ResponseWriter, r *http.Request) {
	user, err := b.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target m.Actor
	if err := b.service.DB().First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var rel m.Relationship
	if err := b.service.DB().Joins("Target").First(&rel, "actor_id = ? and target_id = ?", user.Actor.ID, target.ID).Error; err != nil {
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

	rel.Blocking = true
	if err := b.service.DB().Clauses(clause.OnConflict{UpdateAll: true}).Create(&rel).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeRelationship(&rel))
}

func (b *Blocks) Destroy(w http.ResponseWriter, r *http.Request) {
	user, err := b.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var target m.Actor
	if err := b.service.DB().First(&target, chi.URLParam(r, "id")).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var rel m.Relationship
	if err := b.service.DB().Joins("Target").First(&rel, "actor_id = ? and target_id = ?", user.Actor.ID, target.ID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rel.Blocking = false
	if err := b.service.DB().Clauses(clause.OnConflict{UpdateAll: true}).Create(&rel).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	toJSON(w, serializeRelationship(&rel))
}
