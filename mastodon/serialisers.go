package mastodon

import (
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/media"
)

// serialisers for various mastodon API responses.

type Account struct {
	ID             snowflake.ID     `json:"id,string"`
	Username       string           `json:"username"`
	Acct           string           `json:"acct"`
	DisplayName    string           `json:"display_name"`
	Locked         bool             `json:"locked"`
	Bot            bool             `json:"bot"`
	Discoverable   *bool            `json:"discoverable"`
	Group          bool             `json:"group"`
	CreatedAt      string           `json:"created_at"`
	Note           string           `json:"note"`
	URL            string           `json:"url"`
	Avatar         string           `json:"avatar"`        // these four fields _cannot_ be blank
	AvatarStatic   string           `json:"avatar_static"` // if they are, various clients will consider the
	Header         string           `json:"header"`        // account to be invalid and ignore it or just go weird :grr:
	HeaderStatic   string           `json:"header_static"` // so they must be set to a default image.
	FollowersCount int32            `json:"followers_count"`
	FollowingCount int32            `json:"following_count"`
	StatusesCount  int32            `json:"statuses_count"`
	LastStatusAt   *string          `json:"last_status_at"`
	NoIndex        bool             `json:"noindex"` // default false
	Emojis         []map[string]any `json:"emojis"`
	Fields         []map[string]any `json:"fields"`
}

type CredentialAccount struct {
	*Account
	Source Source `json:"source"`
	Role   *Role  `json:"role,omitempty"`
}

type Role struct {
	ID          uint32 `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Position    int32  `json:"position"`
	Permissions uint32 `json:"permissions"`
	Highlighted bool   `json:"highlighted"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Source struct {
	Privacy             string           `json:"privacy"`
	Sensitive           bool             `json:"sensitive"`
	Language            string           `json:"language"`
	Note                string           `json:"note"`
	FollowRequestsCount int32            `json:"follow_requests_count"`
	Fields              []map[string]any `json:"fields"`
}

func serialiseAccount(a *models.Actor) *Account {
	return &Account{
		ID:             a.ID,
		Username:       a.Name,
		Acct:           a.Acct(),
		DisplayName:    a.DisplayName,
		Locked:         a.Locked,
		Bot:            a.IsBot(),
		Group:          a.IsGroup(),
		CreatedAt:      snowflake.ID(a.ID).ToTime().Round(time.Hour).Format("2006-01-02T00:00:00.000Z"),
		Note:           a.Note,
		URL:            fmt.Sprintf("https://%s/@%s", a.Domain, a.Name),
		Avatar:         media.ProxyAvatarURL(a),
		AvatarStatic:   media.ProxyAvatarURL(a),
		Header:         media.ProxyHeaderURL(a),
		HeaderStatic:   media.ProxyHeaderURL(a),
		FollowersCount: a.FollowersCount,
		FollowingCount: a.FollowingCount,
		StatusesCount:  a.StatusesCount,
		LastStatusAt: func() *string {
			if a.LastStatusAt.IsZero() {
				return nil
			}
			st := a.LastStatusAt.Format("2006-01-02")
			return &st
		}(),
		Emojis: make([]map[string]any, 0), // must be an empty array -- not null
		Fields: make([]map[string]any, 0), // ditto
	}
}

func serialiseCredentialAccount(a *models.Account) *CredentialAccount {
	ca := CredentialAccount{
		Account: serialiseAccount(a.Actor),
		Source: Source{
			Privacy:   "public",
			Sensitive: false,
			Language:  "en",
			Note:      a.Actor.Note,
		},
	}
	if a.Role != nil {
		ca.Role = &Role{
			ID:          a.Role.ID,
			Name:        a.Role.Name,
			Color:       a.Role.Color,
			Position:    a.Role.Position,
			Permissions: a.Role.Permissions,
			Highlighted: a.Role.Highlighted,
			CreatedAt:   a.Role.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
			UpdatedAt:   a.Role.UpdatedAt.Format("2006-01-02T15:04:05.006Z"),
		}
	}
	return &ca
}

type Relationship struct {
	ID                  snowflake.ID `json:"id,string"`
	Following           bool         `json:"following"`
	ShowingReblogs      bool         `json:"showing_reblogs"`
	Notifying           bool         `json:"notifying"`
	FollowedBy          bool         `json:"followed_by"`
	Blocking            bool         `json:"blocking"`
	BlockedBy           bool         `json:"blocked_by"`
	Muting              bool         `json:"muting"`
	MutingNotifications bool         `json:"muting_notifications"`
	Requested           bool         `json:"requested"`
	DomainBlocking      bool         `json:"domain_blocking"`
	Endorsed            bool         `json:"endorsed"`
	Note                string       `json:"note"`
}

func serialiseRelationship(rel *models.Relationship) *Relationship {
	return &Relationship{
		ID:                  rel.TargetID,
		Following:           rel.Following,
		ShowingReblogs:      true,  // todo
		Notifying:           false, // todo
		FollowedBy:          rel.FollowedBy,
		Blocking:            rel.Blocking,
		BlockedBy:           rel.BlockedBy,
		Muting:              rel.Muting,
		MutingNotifications: false,
		Requested:           false,
		DomainBlocking:      false,
		Endorsed:            false,
		Note: func() string {
			// FirstOrCreate won't preload the Target
			// so it will be zero. :(
			if rel.Target == nil {
				return ""
			}
			return rel.Target.Note
		}(),
	}
}

func serialiseStatus(s *models.Status) map[string]any {
	return map[string]any{
		"id":                     toString(s.ID),
		"created_at":             snowflake.ID(s.ID).ToTime().Round(time.Second).Format("2006-01-02T15:04:05.000Z"),
		"edited_at":              nil,
		"in_reply_to_id":         stringOrNull(s.InReplyToID),
		"in_reply_to_account_id": stringOrNull(s.InReplyToActorID),
		"sensitive":              s.Sensitive,
		"spoiler_text":           s.SpoilerText,
		"visibility":             s.Visibility,
		"language":               "en", // s.Language,
		"uri":                    s.URI,
		"url":                    nil,
		"text":                   nil, // not optional!!
		"replies_count":          s.RepliesCount,
		"reblogs_count":          s.ReblogsCount,
		"favourites_count":       s.FavouritesCount,
		"favourited":             s.Reaction != nil && s.Reaction.Favourited,
		"reblogged":              s.Reaction != nil && s.Reaction.Reblogged,
		"muted":                  s.Reaction != nil && s.Reaction.Muted,
		"bookmarked":             s.Reaction != nil && s.Reaction.Bookmarked,
		"content":                s.Note,
		"reblog": func(s *models.Status) any {
			if s.Reblog == nil {
				return nil
			}
			return serialiseStatus(s.Reblog)
		}(s),
		// "filtered":          []map[string]any{},
		"account":           serialiseAccount(s.Actor),
		"media_attachments": serialiseAttachments(s.Attachments),
		"mentions":          []map[string]any{},
		"tags":              []map[string]any{},
		"emojis":            []map[string]any{},
		"card":              nil,
		"poll":              nil,
	}
}

type MediaAttachment struct {
	ID          snowflake.ID   `json:"id,string"`
	Type        string         `json:"type"`
	URL         string         `json:"url"`
	PreviewURL  string         `json:"preview_url"`
	RemoteURL   any            `json:"remote_url"`
	Meta        map[string]any `json:"meta"`
	Description string         `json:"description"`
	Blurhash    string         `json:"blurhash"`
}

func serialiseAttachments(atts []models.StatusAttachment) []MediaAttachment {
	res := []MediaAttachment{} // ensure we return a slice, not null
	for _, att := range atts {
		res = append(res, MediaAttachment{
			ID:         att.ID,
			Type:       attachmentType(&att.Attachment),
			URL:        att.URL,
			PreviewURL: att.URL,
			RemoteURL:  nil,
			Meta: map[string]any{
				"original": map[string]any{
					"width":  att.Width,
					"height": att.Height,
					"size":   fmt.Sprintf("%dx%d", att.Width, att.Height),
					"aspect": float64(att.Width) / float64(att.Height),
				},
				// "small": map[string]any{
				// 	"width":  att.Width,
				// 	"height": att.Height,
				// 	"size":   fmt.Sprintf("%dx%d", att.Attachment.Width, att.Attachment.Height),
				// 	"aspect": float64(att.Attachment.Width) / float64(att.Attachment.Height),
				// },
				// "focus": map[string]any{
				// 	"x": 0.0,
				// 	"y": 0.0,
				// },
			},
			Description: att.Name,
			Blurhash:    att.Blurhash,
		})
	}
	return res
}

func attachmentType(att *models.Attachment) string {
	switch att.MediaType {
	case "image/jpeg":
		return "image"
	case "image/png":
		return "image"
	case "image/gif":
		return "image"
	case "video/mp4":
		return "video"
	case "video/webm":
		return "video"
	case "audio/mpeg":
		return "audio"
	case "audio/ogg":
		return "audio"
	default:
		return "unknown" // todo YOLO
	}
}

func serialiseInstanceV1(i *models.Instance) map[string]any {
	return map[string]any{
		"uri":               i.Domain,
		"title":             i.Title,
		"short_description": stringOrDefault(i.ShortDescription, i.Description),
		"description":       i.Description,
		"email":             i.Admin.Email,
		"version":           "https://github.com/davecheney/pub@0.0.1-devel",
		"urls": map[string]any{
			"streaming_api": "wss://" + i.Domain + "/api/v1/streaming",
		},
		"stats": map[string]any{
			"user_count":   i.AccountsCount,
			"status_count": i.StatusesCount,
			"domain_count": i.DomainsCount,
		},
		"thumbnail":         i.Thumbnail,
		"languages":         []any{"en"},
		"registrations":     false,
		"approval_required": false,
		"invites_enabled":   false,
		"configuration": map[string]any{
			"accounts": map[string]any{
				"max_featured_tags": 4,
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
		"contact_account": serialiseAccount(i.Admin.Actor),
		"rules":           serialiseRules(i),
	}
}

func serialiseInstanceV2(i *models.Instance) map[string]any {
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
				"account": serialiseAccount(i.Admin.Actor),
			},
			"rules": serialiseRules(i),
		},
	}
}

type Rule struct {
	ID   uint32 `json:"id,string"`
	Text string `json:"text"`
}

func serialiseRules(i *models.Instance) []Rule {
	rules := []Rule{}
	for _, rule := range i.Rules {
		rules = append(rules, Rule{
			ID:   rule.ID,
			Text: rule.Text,
		})
	}
	return rules
}

type Marker struct {
	LastReadID snowflake.ID `json:"last_read_id,string"`
	Version    int32        `json:"version"`
	UpdatedAt  string
}

func seraliseMarker(m *models.AccountMarker) *Marker {
	return &Marker{
		LastReadID: m.LastReadID,
		Version:    m.Version,
		UpdatedAt:  m.UpdatedAt.Format("2006-01-02T15:04:05.006Z"),
	}
}

type List struct {
	ID            snowflake.ID `json:"id,string"`
	Title         string       `json:"title"`
	RepliesPolicy string       `json:"replies_policy"`
}

func serialiseList(l *models.AccountList) *List {
	return &List{
		ID:            l.ID,
		Title:         l.Title,
		RepliesPolicy: l.RepliesPolicy,
	}
}

type Application struct {
	ID           snowflake.ID `json:"id,string"`
	Name         string       `json:"name"`
	Website      any          `json:"website,omitempty"` // string or null
	VapidKey     string       `json:"vapid_key"`
	ClientID     string       `json:"client_id,omitempty"`
	ClientSecret string       `json:"client_secret,omitempty"`
}

func serialiseApplication(a *models.Application) *Application {
	return &Application{
		ID:           a.ID,
		Name:         a.Name,
		Website:      a.Website,
		VapidKey:     a.VapidKey,
		ClientID:     a.ClientID,
		ClientSecret: a.ClientSecret,
	}
}
