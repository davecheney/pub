package mastodon

import (
	"fmt"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
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
	Fields         []Field          `json:"fields"`
}

type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	// TODO verified_at
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
		CreatedAt:      a.ID.ToTime().Round(time.Hour).Format("2006-01-02T00:00:00.000Z"),
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
		Fields: algorithms.Map(a.Attributes, func(a *models.ActorAttribute) Field {
			return Field{
				Name:  a.Name,
				Value: a.Value,
			}
		}),
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

// Status is a representation of a Mastodon Status object.
// https://docs.joinmastodon.org/entities/Status/
type Status struct {
	ID                 snowflake.ID       `json:"id,string"`
	CreatedAt          string             `json:"created_at"`
	EditedAt           any                `json:"edited_at"`
	InReplyToID        *snowflake.ID      `json:"in_reply_to_id,string"`
	InReplyToAccountID *snowflake.ID      `json:"in_reply_to_account_id,string"`
	Sensitive          bool               `json:"sensitive"`
	SpoilerText        string             `json:"spoiler_text"`
	Visibility         string             `json:"visibility"`
	Language           string             `json:"language"`
	URI                string             `json:"uri"`
	URL                any                `json:"url"`
	Text               any                `json:"text"`
	RepliesCount       int                `json:"replies_count"`
	ReblogsCount       int                `json:"reblogs_count"`
	FavouritesCount    int                `json:"favourites_count"`
	Favourited         bool               `json:"favourited"`
	Reblogged          bool               `json:"reblogged"`
	Muted              bool               `json:"muted"`
	Bookmarked         bool               `json:"bookmarked"`
	Content            string             `json:"content"`
	Reblog             *Status            `json:"reblog"`
	Account            *Account           `json:"account"`
	MediaAttachments   []*MediaAttachment `json:"media_attachments"`
	Mentions           []*Mention         `json:"mentions"`
	Tags               []*Tag             `json:"tags"`
	Emojis             []any              `json:"emojis"`
	Card               any                `json:"card"`
	Poll               *Poll              `json:"poll"`
	Application        any                `json:"application"`
}

func serialiseStatus(s *models.Status) *Status {
	if s == nil {
		return nil
	}
	createdAt := s.ID.ToTime()
	st := &Status{
		ID:                 s.ID,
		CreatedAt:          createdAt.Round(time.Second).Format("2006-01-02T15:04:05.000Z"),
		EditedAt:           maybeEditedAt(createdAt, s.UpdatedAt),
		InReplyToID:        s.InReplyToID,
		InReplyToAccountID: s.InReplyToActorID,
		Sensitive:          s.Sensitive,
		SpoilerText:        s.SpoilerText,
		Visibility:         s.Visibility,
		Language:           s.Language,
		URI:                s.URI,
		URL:                nil,
		Text:               nil, // not optional!!
		RepliesCount:       s.RepliesCount,
		ReblogsCount:       s.ReblogsCount,
		FavouritesCount:    s.FavouritesCount,
		Favourited:         s.Reaction != nil && s.Reaction.Favourited,
		Reblogged:          s.Reaction != nil && s.Reaction.Reblogged,
		Muted:              s.Reaction != nil && s.Reaction.Muted,
		Bookmarked:         s.Reaction != nil && s.Reaction.Bookmarked,
		Content:            s.Note,
		Reblog:             serialiseStatus(s.Reblog),
		Account:            serialiseAccount(s.Actor),
		MediaAttachments:   algorithms.Map(algorithms.Map(s.Attachments, statusAttachmentToAttachment), serialiseAttachment),
		Mentions:           algorithms.Map(algorithms.Map(s.Mentions, statusMentionToActor), serialiseMention),
		Tags: algorithms.Map(algorithms.Map(s.Tags, statusTagToTag), func(t *models.Tag) *Tag {
			return &Tag{
				Name: t.Name,
				URL:  fmt.Sprintf("/tags/%s", t.Name), // todo, this URL should be absolute to the instance
			}
		}),
		Emojis:      []any{},
		Card:        nil,
		Poll:        serialisePoll(s.Poll),
		Application: nil,
	}
	return st
}

// maybeEditedAt returns a string representation of the time the status was edited, or null if it was not edited.
func maybeEditedAt(createdAt, updatedAt time.Time) any {
	if updatedAt.Equal(createdAt) || updatedAt.Before(createdAt) {
		return nil
	}
	return updatedAt.Round(time.Second).Format("2006-01-02T15:04:05.000Z")
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

func statusAttachmentToAttachment(sa *models.StatusAttachment) *models.Attachment {
	return &sa.Attachment
}

func statusMentionToActor(sm models.StatusMention) *models.Actor {
	return sm.Actor
}

func statusTagToTag(st models.StatusTag) *models.Tag {
	return st.Tag
}

func serialiseAttachment(att *models.Attachment) *MediaAttachment {
	return &MediaAttachment{
		ID:         att.ID,
		Type:       attachmentType(att),
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
	}
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

// https://docs.joinmastodon.org/entities/Marker/
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

// Mention represents a mention of a user in the context of a status.
// https://docs.joinmastodon.org/entities/Status/#Mention
type Mention struct {
	ID       snowflake.ID `json:"id,string"`
	URL      string       `json:"url"`
	Acct     string       `json:"acct"`
	Username string       `json:"username"`
}

func serialiseMention(a *models.Actor) *Mention {
	return &Mention{
		ID:       a.ID,
		URL:      a.URL(),
		Acct:     a.Acct(),
		Username: a.Name,
	}
}

// Tag represents a hashtag in the context of a status.
// https://docs.joinmastodon.org/entities/Tag
type Tag struct {
	Name    string           `json:"name"`
	URL     string           `json:"url"`
	History []map[string]any `json:"history,omitempty"`
}

// https://docs.joinmastodon.org/entities/Poll/
type Poll struct {
	ID          snowflake.ID `json:"id,string"`
	ExpiresAt   string       `json:"expires_at"`
	Expired     bool         `json:"expired"`
	Multiple    bool         `json:"multiple"`
	VotesCount  int          `json:"votes_count"`
	VotersCount any          `json:"voters_count"`
	Voted       bool         `json:"voted"`
	Options     []PollOption `json:"options"`
	Emojies     []any        `json:"emojies"`
}

type PollOption struct {
	Title      string `json:"title"`
	VotesCount int    `json:"votes_count"`
}

func serialisePoll(p *models.StatusPoll) *Poll {
	if p == nil {
		return nil
	}
	return &Poll{
		ID:          p.StatusID,
		ExpiresAt:   p.ExpiresAt.Format("2006-01-02T15:04:05.006Z"),
		Expired:     p.ExpiresAt.After(time.Now()),
		Multiple:    p.Multiple,
		VotesCount:  p.VotesCount,
		VotersCount: nil,
		Voted:       false,
		Options:     serialisePollOptions(p.Options),
		Emojies:     nil,
	}
}

func serialisePollOptions(options []models.StatusPollOption) []PollOption {
	pollOptions := []PollOption{}
	for _, option := range options {
		pollOptions = append(pollOptions, PollOption{
			Title:      option.Title,
			VotesCount: option.Count,
		})
	}
	return pollOptions
}
