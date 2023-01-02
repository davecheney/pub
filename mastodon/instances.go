package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/to"
)

func InstancesIndexV1(env *Env, w http.ResponseWriter, r *http.Request) error {
	instance, err := env.findByDomain(r.Host)
	if err != nil {
		return err
	}
	if err := env.DB.Model(&models.Instance{}).Count(&instance.DomainsCount).Error; err != nil {
		return err
	}
	return to.JSON(w, serialiseInstanceV1(instance))
}

func InstancesIndexV2(env *Env, w http.ResponseWriter, r *http.Request) error {
	instance, err := env.findByDomain(r.Host)
	if err != nil {
		return err
	}
	if err := env.DB.Model(&models.Instance{}).Count(&instance.DomainsCount).Error; err != nil {
		return err
	}
	return to.JSON(w, serialiseInstanceV2(instance))
}

func InstancesPeersShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	var domains []string
	if err := env.DB.Model(&models.Actor{}).Group("Domain").Where("Domain != ?", r.Host).Pluck("domain", &domains).Error; err != nil {
		return err
	}
	return to.JSON(w, domains)
}

func InstancesActivityShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	return to.JSON(w, []map[string]interface{}{})
}

func InstancesDomainBlocksShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	return to.JSON(w, []map[string]interface{}{})
}
