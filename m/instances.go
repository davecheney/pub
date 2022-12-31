package m

import (
	"net/http"

	"github.com/davecheney/m/internal/models"
	"gorm.io/gorm"
)

type instances struct {
	db *gorm.DB
}

func (i *instances) Count() (int, error) {
	var count int64
	if err := i.db.Model(&models.Instance{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// ForRequest returns the instance for the given request.
func (i *instances) ForRequest(r *http.Request) (*models.Instance, error) {
	return i.FindByDomain(r.Host)
}

func (i *instances) FindByDomain(domain string) (*models.Instance, error) {
	var instance models.Instance
	if err := i.db.Where("domain = ?", domain).Preload("Admin").Preload("Admin.Actor").Preload("Rules").First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}
