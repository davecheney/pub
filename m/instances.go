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

// func (i *instances) newRemoteInstanceFetcher() *RemoteInstanceFetcher {
// 	return &RemoteInstanceFetcher{}
// }

// type RemoteInstanceFetcher struct {
// }

// func (r *RemoteInstanceFetcher) Fetch(domain string) (*models.Instance, error) {
// 	obj, err := r.fetch(domain)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &models.Instance{
// 		ID:               snowflake.TimeToID(time.Now()),
// 		Domain:           domain,
// 		Title:            stringFromAny(obj["title"]),
// 		ShortDescription: stringFromAny(obj["short_description"]),
// 		Description:      stringFromAny(obj["description"]),
// 	}, nil
// }

// func (r *RemoteInstanceFetcher) fetch(domain string) (map[string]any, error) {
// 	var obj map[string]any
// 	err := requests.URL("https://" + domain + "/api/v1/instance").ToJSON(&obj).Fetch(context.Background())
// 	if err != nil {
// 		return nil, err
// 	}
// 	return obj, nil
// }

// func (i *instances) FindOrCreate(domain string, fn func(string) (*models.Instance, error)) (*models.Instance, error) {
// 	var instance models.Instance
// 	err := i.db.Where("domain = ?", domain).First(&instance).Error
// 	if err == nil {
// 		return &instance, nil
// 	}
// 	if err != gorm.ErrRecordNotFound {
// 		return nil, err
// 	}
// 	inst, err := fn(domain)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if err := i.db.Create(inst).Error; err != nil {
// 		return nil, err
// 	}
// 	return inst, nil
// }

func stringOrDefault(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}
