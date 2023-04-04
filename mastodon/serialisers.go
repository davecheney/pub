package mastodon

import (
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/media"
)

// Seraliser contains methods to seralise various Mastodon REST API
// responses.
type Serialiser struct {
	req *http.Request
}

type Account struct {
	ID             snowflake.ID `json:"id,string"`
	Username       string       `json:"username"`
	Acct           string       `json:"acct"`
	DisplayName    string       `json:"display_name"`
	Locked         bool         `json:"locked"`
	Bot            bool         `json:"bot"`
	Discoverable   bool         `json:"discoverable"`
	Group          bool         `json:"group"`
	CreatedAt      string       `json:"created_at"`
	Note           string       `json:"note"`
	URL            string       `json:"url"`
	Avatar         string       `json:"avatar"`        // these four fields _cannot_ be blank
	AvatarStatic   string       `json:"avatar_static"` // if they are, various clients will consider the
	Header         string       `json:"header"`        // account to be invalid and ignore it or just go weird :grr:
	HeaderStatic   string       `json:"header_static"` // so they must be set to a default image.
	FollowersCount int32        `json:"followers_count"`
	FollowingCount int32        `json:"following_count"`
	StatusesCount  int32        `json:"statuses_count"`
	LastStatusAt   *string      `json:"last_status_at"`
	// NoIndex        bool             `json:"noindex"` // default false
	Emojis []map[string]any `json:"emojis"`
	Fields []Field          `json:"fields"`
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

type Source struct {
	Privacy             string           `json:"privacy"`
	Sensitive           bool             `json:"sensitive"`
	Language            string           `json:"language"`
	Note                string           `json:"note"`
	FollowRequestsCount int32            `json:"follow_requests_count"`
	Fields              []map[string]any `json:"fields"`
}

func (s *Serialiser) Account(a *models.Actor) *Account {
	return &Account{
		ID:             a.ID,
		Username:       a.Name,
		Acct:           a.Acct(),
		DisplayName:    a.DisplayName,
		Locked:         a.Locked,
		Bot:            a.IsBot(),
		Discoverable:   true, // TODO
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

func (s *Serialiser) CredentialAccount(a *models.Account) *CredentialAccount {
	return &CredentialAccount{
		Account: s.Account(a.Actor),
		Source: Source{
			Privacy:   "public",
			Sensitive: false,
			Language:  "en",
			Note:      a.Actor.Note,
		},
		Role: s.Role(a.Role),
	}
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

func (s *Serialiser) Role(ar *models.AccountRole) *Role {
	if ar == nil {
		return nil
	}
	return &Role{
		ID:          ar.ID,
		Name:        ar.Name,
		Color:       ar.Color,
		Position:    ar.Position,
		Permissions: ar.Permissions,
		Highlighted: ar.Highlighted,
		CreatedAt:   ar.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
		UpdatedAt:   ar.UpdatedAt.Format("2006-01-02T15:04:05.006Z"),
	}
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

func (s *Serialiser) Relationship(rel *models.Relationship) *Relationship {
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
	InReplyToID        *snowflake.ID      `json:"in_reply_to_id,string"`
	InReplyToAccountID *snowflake.ID      `json:"in_reply_to_account_id,string"`
	Sensitive          bool               `json:"sensitive"`
	SpoilerText        string             `json:"spoiler_text"`
	Visibility         models.Visibility  `json:"visibility"`
	Language           any                `json:"language"`
	URI                string             `json:"uri"`
	URL                any                `json:"url"`
	RepliesCount       int                `json:"replies_count"`
	ReblogsCount       int                `json:"reblogs_count"`
	FavouritesCount    int                `json:"favourites_count"`
	EditedAt           any                `json:"edited_at"`
	Favourited         bool               `json:"favourited"`
	Reblogged          bool               `json:"reblogged"`
	Muted              bool               `json:"muted"`
	Bookmarked         bool               `json:"bookmarked"`
	Content            string             `json:"content"`
	Filtered           []any              `json:"filtered"`
	Reblog             *Status            `json:"reblog"`
	Application        any                `json:"application,omitempty"`
	Account            *Account           `json:"account"`
	MediaAttachments   []*MediaAttachment `json:"media_attachments"`
	Mentions           []*Mention         `json:"mentions"`
	Tags               []*Tag             `json:"tags"`
	Emojis             []any              `json:"emojis"`
	Card               any                `json:"card"`
	Poll               *Poll              `json:"poll"`
}

func (s *Serialiser) Status(st *models.Status) *Status {
	if st == nil {
		return nil
	}
	createdAt := st.ID.ToTime()
	return &Status{
		ID:                 st.ID,
		CreatedAt:          createdAt.Round(time.Second).Format("2006-01-02T15:04:05.000Z"),
		EditedAt:           maybeEditedAt(createdAt, st.UpdatedAt),
		InReplyToID:        st.InReplyToID,
		InReplyToAccountID: st.InReplyToActorID,
		Sensitive:          st.Sensitive,
		SpoilerText:        st.SpoilerText,
		Visibility: func() models.Visibility {
			if st.Visibility == "limited" {
				return "private"
			}
			return st.Visibility
		}(),
		Language: func() any {
			if st.Reblog != nil {
				return nil
			}
			if st.Language == "" {
				return "en"
			}
			return st.Language
		}(),
		URI: st.URI,
		URL: func() any {
			if st.Reblog != nil {
				return nil
			}
			return st.URI
		}(),
		RepliesCount:     st.RepliesCount,
		ReblogsCount:     st.ReblogsCount,
		FavouritesCount:  st.FavouritesCount,
		Favourited:       st.Reaction != nil && st.Reaction.Favourited,
		Reblogged:        st.Reaction != nil && st.Reaction.Reblogged,
		Muted:            st.Reaction != nil && st.Reaction.Muted,
		Bookmarked:       st.Reaction != nil && st.Reaction.Bookmarked,
		Content:          st.Note,
		Reblog:           s.Status(st.Reblog),
		Account:          s.Account(st.Actor),
		MediaAttachments: s.MediaAttachments(st.Attachments),
		Mentions:         s.Mentions(st.Mentions),
		Tags:             s.Tags(st.Tags),
		Emojis:           nil,
		Card:             nil,
		Poll:             s.Poll(st.Poll),
	}
}

func (s *Serialiser) Tags(tags []models.StatusTag) []*Tag {
	return algorithms.Map(
		algorithms.Map(
			tags,
			func(st models.StatusTag) *models.Tag {
				return st.Tag
			},
		),
		func(t *models.Tag) *Tag {
			return &Tag{
				Name: t.Name,
				URL:  s.urlFor("/tags/" + t.Name),
			}
		},
	)
}

func (s *Serialiser) Mentions(mentions []models.StatusMention) []*Mention {
	return algorithms.Map(
		algorithms.Map(
			mentions,
			func(sm models.StatusMention) *models.Actor {
				return sm.Actor
			},
		),
		func(a *models.Actor) *Mention {
			return &Mention{
				ID:       a.ID,
				URL:      a.URL(),
				Acct:     a.Acct(),
				Username: a.Name,
			}
		},
	)
}

func (s *Serialiser) urlFor(path string) string {
	return fmt.Sprintf("https://%s%s", s.req.Host, path)
}

// nilIfEmpty returns nil if the string is empty, otherwise returns the string.
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// maybeEditedAt returns a string representation of the time the status was edited, or null if it was not edited.
func maybeEditedAt(createdAt, updatedAt time.Time) any {
	if updatedAt.Equal(createdAt) || updatedAt.Before(createdAt) {
		return nil
	}
	return updatedAt.Round(time.Second).Format("2006-01-02T15:04:05.000Z")
}

type MediaAttachment struct {
	ID               snowflake.ID `json:"id,string"`
	Type             string       `json:"type"`
	URL              string       `json:"url"`
	PreviewURL       string       `json:"preview_url,omitempty"`
	RemoteURL        any          `json:"remote_url,omitempty"`
	PreviewRemoteURL any          `json:"preview_remote_url,omitempty"`
	TextURL          any          `json:"text_url,omitempty"`
	Meta             Meta         `json:"meta"`
	Description      string       `json:"description,omitempty"`
	Blurhash         string       `json:"blurhash,omitempty"`
}

type Meta struct {
	Original      *MetaFormat `json:"original,omitempty"`
	Small         *MetaFormat `json:"small,omitempty"`
	Focus         MetaFocus   `json:"focus,omitempty"`
	Length        string      `json:"length,omitempty"`
	Duration      float64     `json:"duration,omitzero"`
	FPS           int         `json:"fps,omitzero"`
	Size          string      `json:"size,omitempty"`
	Width         int         `json:"width,omitzero"`
	Height        int         `json:"height,omitzero"`
	Aspect        float64     `json:"aspect,omitzero"`
	AudioEncode   string      `json:"audio_encode,omitempty"`
	AudioBitrate  string      `json:"audio_bitrate,omitempty"`
	AudioChannels string      `json:"audio_channels,omitempty"`
}

type MetaFocus struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type MetaFormat struct {
	Width     int     `json:"width,omitempty"`
	Height    int     `json:"height,omitempty"`
	Size      string  `json:"size,omitempty"`
	Aspect    float64 `json:"aspect,omitzero"`
	FrameRate string  `json:"frame_rate,omitempty"`
	Duration  float64 `json:"duration,omitzero"`
	Bitrate   string  `json:"bitrate,omitempty"`
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
		return "unknown"
	}
}

func extension(att *models.Attachment) string {
	switch att.MediaType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "video/mp4":
		return "mp4"
	case "video/webm":
		return "webm"
	case "audio/mpeg":
		return "mp3"
	case "audio/ogg":
		return "ogg"
	default:
		return "jpg" // todo YOLO
	}
}

func (s *Serialiser) InstanceV1(i *models.Instance) map[string]any {
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
		"contact_account": s.Account(i.Admin.Actor),
		"rules":           s.Rules(i),
	}
}

func (s *Serialiser) InstanceV2(i *models.Instance) map[string]any {
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
				"account": s.Account(i.Admin.Actor),
			},
			"rules": s.Rules(i),
		},
	}
}

type Rule struct {
	ID   uint32 `json:"id,string"`
	Text string `json:"text"`
}

func (s *Serialiser) Rules(i *models.Instance) []Rule {
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

func (s *Serialiser) Marker(m *models.AccountMarker) *Marker {
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

func (s *Serialiser) List(l *models.AccountList) *List {
	return &List{
		ID:            l.ID,
		Title:         l.Title,
		RepliesPolicy: l.RepliesPolicy,
	}
}

type Application struct {
	ID           snowflake.ID `json:"id,string"`
	Name         string       `json:"name"`
	Website      any          `json:"website"` // string or null
	RedirectURI  string       `json:"redirect_uri,omitempty"`
	VapidKey     string       `json:"vapid_key"`
	ClientID     string       `json:"client_id,omitempty"`
	ClientSecret string       `json:"client_secret,omitempty"`
}

func (s *Serialiser) Application(a *models.Application) *Application {
	return &Application{
		ID:           a.ID,
		Name:         a.Name,
		Website:      nilIfEmpty(a.Website),
		RedirectURI:  a.RedirectURI,
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

func (s *Serialiser) Poll(p *models.StatusPoll) *Poll {
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
		Options: algorithms.Map(
			p.Options,
			func(option models.StatusPollOption) PollOption {
				return PollOption{
					Title:      option.Title,
					VotesCount: option.Count,
				}
			},
		),
	}
}

// https://docs.joinmastodon.org/entities/StatusEdit/
type StatusEdit struct {
	Content          string             `json:"content"`
	SpoilerText      string             `json:"spoiler_text"`
	Sensitive        bool               `json:"sensitive"`
	CreatedAt        string             `json:"created_at"` // updated_at if edited, created_at if original
	Account          *Account           `json:"account"`
	Poll             *Poll              `json:"poll"`
	MediaAttachments []*MediaAttachment `json:"media_attachments"`
	Emojies          []any              `json:"emojies"`
}

func (s *Serialiser) StatusEdit(st *models.Status) *StatusEdit {
	return &StatusEdit{
		Content:     st.Note,
		SpoilerText: st.SpoilerText,
		Sensitive:   st.Sensitive,
		CreatedAt: func() time.Time {
			createdAt := st.ID.ToTime()
			if st.UpdatedAt.After(createdAt) {
				return st.UpdatedAt
			}
			return createdAt
		}().Format("2006-01-02T15:04:05.006Z"),
		Account:          s.Account(st.Actor),
		Poll:             s.Poll(st.Poll),
		MediaAttachments: s.MediaAttachments(st.Attachments),
		Emojies:          nil,
	}
}

const (
	PREVIEW_MAX_WIDTH  = 560
	PREVIEW_MAX_HEIGHT = 415
)

func (s *Serialiser) MediaAttachments(attachments []*models.StatusAttachment) []*MediaAttachment {
	return algorithms.Map(
		algorithms.Map(
			attachments,
			func(sa *models.StatusAttachment) *models.Attachment {
				return &sa.Attachment
			},
		), func(att *models.Attachment) *MediaAttachment {
			at := &MediaAttachment{
				ID:         att.ID,
				Type:       attachmentType(att),
				URL:        s.mediaOriginalURL(att),
				PreviewURL: s.mediaPreviewURL(att),
				RemoteURL:  att.URL,
				Meta: Meta{
					Focus: MetaFocus{
						X: 0.0, // always centered
						Y: 0.0,
					},
					Original: s.originalMetaFormat(att),
					Small:    s.smallMetaFormat(att),
				},
				Description: att.Name,
				Blurhash:    att.Blurhash,
			}
			return at
		},
	)
}

func (s *Serialiser) originalMetaFormat(att *models.Attachment) *MetaFormat {
	f := &MetaFormat{
		Width:  att.Width,
		Height: att.Height,
		Size:   fmt.Sprintf("%dx%d", att.Width, att.Height),
	}
	if att.Width > 0 && att.Height > 0 {
		f.Aspect = float64(att.Width) / float64(att.Height)
	}
	return f
}

func (s *Serialiser) smallMetaFormat(att *models.Attachment) *MetaFormat {
	if att.Width < PREVIEW_MAX_WIDTH && att.Height < PREVIEW_MAX_HEIGHT {
		// no preview needed
		return s.originalMetaFormat(att)
	}
	switch att.MediaType {
	case "image/jpeg", "image/png", "image/gif":
		h := att.Height
		w := att.Width

		if w > h {
			h = int(float64(h) * (PREVIEW_MAX_WIDTH / float64(w)))
			w = 560
		} else {
			w = int(float64(w) * (PREVIEW_MAX_HEIGHT / float64(h)))
			h = 415
		}

		f := &MetaFormat{
			Width:  w,
			Height: h,
			Size:   fmt.Sprintf("%dx%d", w, h),
		}
		if att.Width > 0 && att.Height > 0 {
			f.Aspect = float64(att.Width) / float64(att.Height)
		}
		return f
	default:
		// no preview needed
		return nil
	}
}

func (s *Serialiser) mediaOriginalURL(att *models.Attachment) string {
	switch att.MediaType {
	case "image/jpeg", "image/png", "image/gif":
		// call through /media proxy to cache
		return s.urlFor(fmt.Sprintf("/media/original/%d.%s", att.ID, extension(att)))
	default:
		// otherwise return the remote URL
		return att.URL
	}
}

func (s *Serialiser) mediaPreviewURL(att *models.Attachment) string {
	if att.Width < PREVIEW_MAX_WIDTH || att.Height < PREVIEW_MAX_HEIGHT {
		// no preview needed
		return ""
	}
	switch att.MediaType {
	case "image/jpeg", "image/png", "image/gif":
		// call through /media proxy to cache
		return s.urlFor(fmt.Sprintf("/media/preview/%d.%s", att.ID, extension(att)))
	default:
		// no preview available
		return ""
	}
}

type Preferences struct {
	PostingDefaultVisibility string `json:"posting:default:visibility"`
	PostingDefaultSensitive  bool   `json:"posting:default:sensitive"`
	PostingDefaultLanguage   any    `json:"posting:default:language"`
	ReadingExpandMedia       string `json:"reading:expand:media"`
	ReadingExpandSpoilers    bool   `json:"reading:expand:spoilers"`
}

func (s *Serialiser) Preferences(prefs *models.AccountPreferences) *Preferences {
	return &Preferences{
		PostingDefaultVisibility: prefs.PostingDefaultVisibility,
		PostingDefaultSensitive:  prefs.PostingDefaultSensitive,
		PostingDefaultLanguage:   nilIfEmpty(prefs.PostingDefaultLanguage),
		ReadingExpandMedia:       prefs.ReadingExpandMedia,
		ReadingExpandSpoilers:    prefs.ReadingExpandSpoilers,
	}
}
