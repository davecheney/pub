package mastodon

import (
	"net/http"
	"strconv"

	"github.com/davecheney/m/m"
	"github.com/go-json-experiment/json"
)

type Instances struct {
	service *Service
}

func (i *Instances) IndexV1(w http.ResponseWriter, r *http.Request) {
	instance, err := i.instanceForHost(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	instance.DomainsCount, err = i.service.Service.Instances().Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeV1(instance))
}

func (i *Instances) IndexV2(w http.ResponseWriter, r *http.Request) {
	instance, err := i.instanceForHost(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	instance.DomainsCount, err = i.service.Service.Instances().Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toJSON(w, serializeV2(instance))
}

func (i *Instances) PeersShow(w http.ResponseWriter, r *http.Request) {
	var instances []m.Instance
	if err := i.service.DB().Model(&m.Instance{}).Preload("Admin").Where("domain != ?", r.Host).Find(&instances).Error; err != nil {
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

func (i *Instances) instanceForHost(r *http.Request) (*m.Instance, error) {
	host := r.Host
	return i.service.Service.Instances().FindByDomain(host)
}

func serializeV1(i *m.Instance) map[string]any {
	return map[string]any{
		"uri":               i.Domain,
		"title":             i.Title,
		"short_description": stringOrDefault(i.ShortDescription, i.Description),
		"description":       i.Description,
		"email":             i.Admin.Email,
		"version":           "https://github.com/davecheney/m@latest",
		"urls":              map[string]any{},
		"stats": map[string]any{
			"user_count":   i.AccountsCount,
			"status_count": i.StatusesCount,
			"domain_count": i.DomainsCount,
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
		"contact_account": serialize(i.Admin.Actor),
		"rules":           serialiseRules(i),
	}
}

func serializeV2(i *m.Instance) map[string]any {
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
				"account": serialize(i.Admin.Actor),
			},
			"rules": serialiseRules(i),
		},
	}
}

func serialiseRules(i *m.Instance) []map[string]any {
	rules := make([]map[string]any, len(i.Rules))
	for i, rule := range i.Rules {
		rules[i] = map[string]any{
			"id":   strconv.Itoa(int(rule.ID)),
			"text": rule.Text,
		}
	}
	return rules
}
