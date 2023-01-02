package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/to"
	"gorm.io/gorm"
)

type Instances struct {
	service *Service
}

func (i *Instances) IndexV1(w http.ResponseWriter, r *http.Request) {
	instance, err := i.forRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	instance.DomainsCount, err = i.count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	to.JSON(w, serialiseInstanceV1(instance))
}

func (i *Instances) IndexV2(w http.ResponseWriter, r *http.Request) {
	instance, err := i.forRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	instance.DomainsCount, err = i.count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	to.JSON(w, serialiseInstanceV2(instance))
}

func InstancesPeersShow(w http.ResponseWriter, r *http.Request) {
	db, _ := r.Context().Value("DB").(*gorm.DB)
	var domains []string
	if err := db.Model(&models.Actor{}).Group("Domain").Where("Domain != ?", r.Host).Pluck("domain", &domains).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	to.JSON(w, domains)
}

func (i *Instances) ActivityShow(w http.ResponseWriter, r *http.Request) {
	to.JSON(w, []map[string]interface{}{})
}

func (i *Instances) DomainBlocksShow(w http.ResponseWriter, r *http.Request) {
	to.JSON(w, []map[string]interface{}{})
}

// Count returns the number of instances in the database.
func (i *Instances) count() (int, error) {
	var count int64
	if err := i.service.db.Model(&models.Instance{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// forRequest returns the instance for the given request.
func (i *Instances) forRequest(r *http.Request) (*models.Instance, error) {
	return i.findByDomain(r.Host)
}

func (i *Instances) findByDomain(domain string) (*models.Instance, error) {
	var instance models.Instance
	if err := i.service.db.Where("domain = ?", domain).Preload("Admin").Preload("Admin.Actor").Preload("Rules").First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}
