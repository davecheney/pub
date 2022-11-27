package mastodon

import (
	"net/http"
	"strconv"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Instance struct {
	gorm.Model
	Domain           string `gorm:"uniqueIndex;size:64"`
	AdminID          *uint
	Admin            *Account
	SourceURL        string
	Title            string `gorm:"size:64"`
	ShortDescription string
	Description      string
	Thumbnail        string `gorm:"size:64"`

	Rules    []InstanceRule `gorm:"foreignKey:InstanceID"`
	Accounts []Account      `gorm:"foreignKey:Domain;references:Domain"`
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

type InstanceRule struct {
	gorm.Model
	InstanceID uint
	Text       string
}

func (i *Instance) serializeV1() map[string]any {
	return map[string]any{
		"uri":               i.Domain,
		"title":             i.Title,
		"short_description": i.ShortDescription,
		"description":       i.Description,
		"email":             i.Admin.Email,
		"version":           "3.5.3",
		"urls":              map[string]any{},
		"stats": map[string]any{
			"user_count":   0,
			"status_count": 0,
			"domain_count": 0,
		},
		"thumbnail":         stringOrDefault(i.Thumbnail, "https://files.mastodon.social/site_uploads/files/000/000/001/original/vlcsnap-2018-08-27-16h43m11s127.png"),
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
				"email":   i.Admin.Email,
				"account": i.Admin.serialize(),
			},
			"rules": i.serialiseRules(),
		},
	}
}

type Instances struct {
	db     *gorm.DB
	domain string
}

func NewInstances(db *gorm.DB, domain string) *Instances {
	return &Instances{
		db:     db,
		domain: domain,
	}
}

func (i *Instances) IndexV1(w http.ResponseWriter, r *http.Request) {
	var instance Instance
	if err := i.db.Where("domain = ?", i.domain).First(&instance).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := i.db.Where("id = ?", instance.AdminID).First(&instance.Admin).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, instance.serializeV1())
}

func (i *Instances) IndexV2(w http.ResponseWriter, r *http.Request) {
	var instance Instance
	if err := i.db.Model(&Instance{}).Preload("Admin").Where("domain = ?", i.domain).First(&instance).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := i.db.Where("id = ?", instance.AdminID).First(&instance.Admin).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, instance.serializeV2())
}

func (i *Instances) Peers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, []string{})
}

func (i *Instances) FindOrCreateInstance(domain string) (*Instance, error) {
	var instance Instance
	if err := i.db.Where("domain = ?", domain).First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			instance = Instance{
				Domain: domain,
			}
			if err := i.db.Create(&instance).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &instance, nil
}

func stringOrDefault(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}
