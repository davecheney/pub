package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

func InstancesIndexV1(env *Env, w http.ResponseWriter, r *http.Request) error {
	return instancesIndex(env, w, r, func(i *models.Instance) map[string]any {
		return map[string]any{
			"uri":               i.Domain,
			"title":             i.Title,
			"short_description": stringOrDefault(i.ShortDescription, i.Description),
			"description":       i.Description,
			"email":             i.Admin.Email,
			"version":           i.SourceURL,
			"urls":              urls(i),
			"stats": map[string]any{
				"user_count":   i.AccountsCount,
				"status_count": i.StatusesCount,
				"domain_count": i.DomainsCount,
			},
			"thumbnail":         i.Thumbnail,
			"languages":         languages(),
			"registrations":     false,
			"approval_required": false,
			"invites_enabled":   false,
			"configuration": map[string]any{
				"accounts": map[string]any{
					"max_featured_tags": 4,
				},
				"statuses":          statuses(),
				"media_attachments": mediaAttachments(),
				"polls":             polls(),
			},
			"contact_account": (&Serialiser{req: r}).Account(i.Admin.Actor),
			"rules":           rules(i),
		}
	})
}

func InstancesIndexV2(env *Env, w http.ResponseWriter, r *http.Request) error {
	return instancesIndex(env, w, r, func(i *models.Instance) map[string]any {
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
			"thumbnail": i.Thumbnail,
			"languages": languages(),
			"configuration": map[string]any{
				"urls": map[string]any{
					"accounts": map[string]any{
						"max_featured_tags": 10,
					},
					"statuses":          statuses(),
					"media_attachments": mediaAttachments(),
					"polls":             polls(),
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
					"account": (&Serialiser{req: r}).Account(i.Admin.Actor),
				},
				"rules": rules(i),
			},
		}
	})
}

func instancesIndex(env *Env, w http.ResponseWriter, r *http.Request, serialiser func(*models.Instance) map[string]any) error {
	var instance models.Instance
	if err := env.DB.Where("domain = ?", r.Host).Preload("Admin").Preload("Admin.Actor").Preload("Rules").Take(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	return to.JSON(w, serialiser(&instance))
}

func InstancesPeersShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	var domains []string
	if err := env.DB.Model(&models.Peer{}).Group("Domain").Where("Domain != ?", r.Host).Pluck("domain", &domains).Error; err != nil {
		return err
	}
	return to.JSON(w, domains)
}

func InstancesRulesShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	var instance models.Instance
	if err := env.DB.Where("domain = ?", r.Host).Preload("Rules").Take(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.Error(http.StatusNotFound, err)
		}
		return err
	}
	return to.JSON(w, rules(&instance))
}

func languages() any {
	return []string{"en"}
}

func mediaAttachments() any {
	return map[string]any{
		"supported_mime_types":   supportedMimeTypes(),
		"image_size_limit":       10485760,
		"image_matrix_limit":     16777216,
		"video_size_limit":       41943040,
		"video_frame_rate_limit": 60,
		"video_matrix_limit":     2304000,
	}
}

func polls() any {
	return map[string]any{
		"max_options":               4,
		"max_characters_per_option": 50,
		"min_expiration":            300,
		"max_expiration":            2629746,
	}
}

func rules(i *models.Instance) any {
	return algorithms.Map(i.Rules, func(r models.InstanceRule) map[string]any {
		return map[string]any{
			"id":   r.ID,
			"text": r.Text,
		}
	})
}

func statuses() any {
	return map[string]any{
		"max_characters":              500,
		"max_media_attachments":       4,
		"characters_reserved_per_url": 23,
	}
}

func supportedMimeTypes() any {
	return []string{
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
	}
}

func urls(i *models.Instance) any {
	return map[string]any{
		"streaming_api": "wss://" + i.Domain + "/api/v1/streaming",
	}
}

func InstancesActivityShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	return to.JSON(w, []map[string]interface{}{})
}

func InstancesDomainBlocksShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	return to.JSON(w, []map[string]interface{}{})
}
