package mastodon

import (
	"net/http"

	"github.com/davecheney/m/internal/models"
)

type Instances struct {
	service *Service
}

func (i *Instances) IndexV1(w http.ResponseWriter, r *http.Request) {
	instance, err := i.service.Service.Instances().ForRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	instance.DomainsCount, err = i.service.Service.Instances().Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialiseInstanceV1(instance))
}

func (i *Instances) IndexV2(w http.ResponseWriter, r *http.Request) {
	instance, err := i.service.Service.Instances().ForRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	instance.DomainsCount, err = i.service.Service.Instances().Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serialiseInstanceV2(instance))
}

func (i *Instances) PeersShow(w http.ResponseWriter, r *http.Request) {
	var domains []string
	if err := i.service.DB().Model(&models.Actor{}).Group("Domain").Where("Domain != ?", r.Host).Pluck("domain", &domains).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, domains)
}

func (i *Instances) ActivityShow(w http.ResponseWriter, r *http.Request) {
	toJSON(w, []map[string]interface{}{})
}

func (i *Instances) DomainBlocksShow(w http.ResponseWriter, r *http.Request) {
	toJSON(w, []map[string]interface{}{})
}
