package m

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Instance struct {
	ID               uint `gorm:"primarykey"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Domain           string `gorm:"uniqueIndex;size:64"`
	AdminID          *uint
	Admin            *Account
	SourceURL        string
	Title            string `gorm:"size:64"`
	ShortDescription string
	Description      string
	Thumbnail        string `gorm:"size:64"`
	AccountsCount    int    `gorm:"default:0;not null"`
	StatusesCount    int    `gorm:"default:0;not null"`

	domainsCount int // not stored

	Rules    []InstanceRule `gorm:"foreignKey:InstanceID"`
	Accounts []Account
}

func (i *Instance) AfterCreate(tx *gorm.DB) error {
	return i.updateAccountsCount(tx)
}

func (i *Instance) serialiseRules() []map[string]any {
	rules := make([]map[string]any, len(i.Rules))
	for i, rule := range i.Rules {
		rules[i] = map[string]any{
			"id":   strconv.Itoa(int(rule.ID)),
			"text": rule.Text,
		}
	}
	return rules
}

func (i *Instance) serializeNodeInfo() map[string]any {
	return map[string]any{
		"version": "2.0", // https://github.com/jhass/nodeinfo/blob/main/schemas/2.0/schema.json
		"software": map[string]any{
			"name":    "https://github.com/davecheney/m",
			"version": "0.0.0-devel",
		},
		"protocols": []string{
			"activitypub",
		},
		"services": map[string]any{
			"outbound": []any{},
			"inbound":  []any{},
		},
		"usage": map[string]any{
			"users": map[string]any{
				"total":          i.AccountsCount,
				"activeMonth":    0,
				"activeHalfyear": 0,
			},
			"localPosts": i.StatusesCount,
		},
		"openRegistrations": false,
		"metadata":          map[string]any{},
	}
}

func (i *Instance) updateAccountsCount(tx *gorm.DB) error {
	var count int64
	err := tx.Model(&Account{}).Where("instance_id = ?", i.ID).Count(&count).Error
	if err != nil {
		return err
	}
	return tx.Model(i).Update("accounts_count", count).Error
}

func (i *Instance) updateStatusesCount(tx *gorm.DB) error {
	var count int64
	err := tx.Model(&Status{}).Joins("Account").Where("instance_id = ?", i.ID).Count(&count).Error
	if err != nil {
		return err
	}
	return tx.Model(i).Update("statuses_count", count).Error
}

type InstanceRule struct {
	gorm.Model
	InstanceID uint
	Text       string
}

func (i *Instance) serializeV1() map[string]any {
	return map[string]any{
		"uri":               i.Domain,
		"title":             i.Title,
		"short_description": stringOrDefault(i.ShortDescription, i.Description),
		"description":       i.Description,
		"email":             i.Admin.LocalAccount.Email,
		"version":           "https://github.com/davecheney/m@latest",
		"urls":              map[string]any{},
		"stats": map[string]any{
			"user_count":   i.AccountsCount,
			"status_count": i.StatusesCount,
			"domain_count": i.domainsCount,
		},
		"thumbnail":         i.Thumbnail,
		"languages":         []any{"en"},
		"registrations":     false,
		"approval_required": false,
		"invites_enabled":   true,
		"configuration": map[string]any{
			"statuses": map[string]any{
				"max_characters":              500,
				"max_media_attachments":       4,
				"characters_reserved_per_url": 23,
			},
			"media_attachments": map[string]any{
				"supported_mime_types": []string{
					"image/jpeg",
					"image/png",
					"image/gif",
					"image/webp",
					"video/webm",
					"video/mp4",
					"video/quicktime",
					"video/ogg",
					"audio/wave",
					"audio/wav",
					"audio/x-wav",
					"audio/x-pn-wave",
					"audio/vnd.wave",
					"audio/ogg",
					"audio/vorbis",
					"audio/mpeg",
					"audio/mp3",
					"audio/webm",
					"audio/flac",
					"audio/aac",
					"audio/m4a",
					"audio/x-m4a",
					"audio/mp4",
					"audio/3gpp",
					"video/x-ms-asf",
				},
				"image_size_limit":       10485760,
				"image_matrix_limit":     16777216,
				"video_size_limit":       41943040,
				"video_frame_rate_limit": 60,
				"video_matrix_limit":     2304000,
			},
			"polls": map[string]any{
				"max_options":               4,
				"max_characters_per_option": 50,
				"min_expiration":            300,
				"max_expiration":            2629746,
			},
		},
		"contact_account": i.Admin.serialize(),
		"rules":           i.serialiseRules(),
	}
}

func (i *Instance) serializeV2() map[string]any {
	return map[string]any{
		"domain":      i.Domain,
		"title":       i.Title,
		"version":     "4.0.0rc1",
		"source_url":  i.SourceURL,
		"description": i.Description,
		"usage": map[string]any{
			"users": map[string]any{
				"active_month": 0,
			},
		},
		"thumbnail": map[string]any{},
		"languages": []any{"en"},
		"configuration": map[string]any{
			"urls": map[string]any{
				"accounts": map[string]any{
					"max_featured_tags": 10,
				},
				"statuses": map[string]any{
					"max_characters":              500,
					"max_media_attachments":       4,
					"characters_reserved_per_url": 23,
				},
				"media_attachments": map[string]any{
					"supported_mime_types": []string{
						"image/jpeg",
						"image/png",
						"image/gif",
						"image/heic",
						"image/heif",
						"image/webp",
						"video/webm",
						"video/mp4",
						"video/quicktime",
						"video/ogg",
						"audio/wave",
						"audio/wav",
						"audio/x-wav",
						"audio/x-pn-wave",
						"audio/vnd.wave",
						"audio/ogg",
						"audio/vorbis",
						"audio/mpeg",
						"audio/mp3",
						"audio/webm",
						"audio/flac",
						"audio/aac",
						"audio/m4a",
						"audio/x-m4a",
						"audio/mp4",
						"audio/3gpp",
						"video/x-ms-asf",
					},
					"image_size_limit":       10485760,
					"image_matrix_limit":     16777216,
					"video_size_limit":       41943040,
					"video_frame_rate_limit": 60,
					"video_matrix_limit":     2304000,
				},
				"polls": map[string]any{
					"max_options":               4,
					"max_characters_per_option": 50,
					"min_expiration":            300,
					"max_expiration":            2629746,
				},
				"translation": map[string]any{
					"enabled": false,
				},
			},
			"registrations": map[string]any{
				"enabled":           false,
				"approval_required": false,
				"message":           nil,
			},
			"contact": map[string]any{
				"email":   i.Admin.LocalAccount.Email,
				"account": i.Admin.serialize(),
			},
			"rules": i.serialiseRules(),
		},
	}
}

type Instances struct {
	db      *gorm.DB
	service *Service
}

func (i *Instances) IndexV1(w http.ResponseWriter, r *http.Request) {
	instance, err := i.instanceForHost(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	instance.domainsCount, err = i.service.instances().Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, instance.serializeV1())
}

func (i *Instances) IndexV2(w http.ResponseWriter, r *http.Request) {
	instance, err := i.instanceForHost(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	instance.domainsCount, err = i.service.instances().Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, instance.serializeV2())
}

func (i *Instances) PeersShow(w http.ResponseWriter, r *http.Request) {
	var instances []Instance
	if err := i.db.Model(&Instance{}).Preload("Admin").Where("domain != ?", r.Host).Find(&instances).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []string
	for _, instance := range instances {
		resp = append(resp, instance.Domain)
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, resp)
}

func (i *Instances) instanceForHost(r *http.Request) (*Instance, error) {
	host := r.Host
	return i.service.instances().FindByDomain(host)
}

type instances struct {
	db *gorm.DB
}

func (i *instances) Count() (int, error) {
	var count int64
	if err := i.db.Model(&Instance{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (i *instances) FindByDomain(domain string) (*Instance, error) {
	var instance Instance
	if err := i.db.Model(&Instance{}).Preload("Admin").Preload("Admin.LocalAccount").Where("domain = ?", domain).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (i *instances) newRemoteInstanceFetcher() *RemoteInstanceFetcher {
	return &RemoteInstanceFetcher{}
}

type RemoteInstanceFetcher struct {
}

func (r *RemoteInstanceFetcher) Fetch(domain string) (*Instance, error) {
	obj, err := r.fetch(domain)
	if err != nil {
		return nil, err
	}
	return &Instance{
		Domain:           domain,
		Title:            stringFromAny(obj["title"]),
		ShortDescription: stringFromAny(obj["short_description"]),
		Description:      stringFromAny(obj["description"]),
	}, nil
}

func (r *RemoteInstanceFetcher) fetch(domain string) (map[string]any, error) {
	var obj map[string]any
	err := requests.URL("https://" + domain + "/api/v1/instance").ToJSON(&obj).Fetch(context.Background())
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (i *instances) FindOrCreate(domain string, fn func(string) (*Instance, error)) (*Instance, error) {
	var instance Instance
	err := i.db.Where("domain = ?", domain).First(&instance).Error
	if err == nil {
		return &instance, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	inst, err := fn(domain)
	if err != nil {
		return nil, err
	}
	if err := i.db.Create(inst).Error; err != nil {
		return nil, err
	}
	return inst, nil
}

func stringOrDefault(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}
