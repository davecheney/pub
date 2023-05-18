package mastodon

import (
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/models"
)

// Seraliser contains methods to seralise various Mastodon REST API
// responses.
type Serialiser struct {
	req *http.Request
}

func NewSerialiser(req *http.Request) Serialiser {
	return Serialiser{req: req}
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
		ID:             a.ObjectID,
		Username:       a.Name,
		Acct:           a.Acct(),
		DisplayName:    a.DisplayName(),
		Locked:         a.Locked(),
		Bot:            a.IsBot(),
		Discoverable:   true, // TODO
		Group:          a.IsGroup(),
		CreatedAt:      a.ObjectID.ToTime().Round(time.Hour).Format("2006-01-02T00:00:00.000Z"),
		Note:           a.Note(),
		URL:            fmt.Sprintf("https://%s/@%s", a.Domain, a.Name),
		Avatar:         stringOrDefault(a.Avatar(), s.urlFor("/avatar.jpg")),
		AvatarStatic:   stringOrDefault(a.Avatar(), s.urlFor("/avatar.jpg")),
		Header:         stringOrDefault(a.Header(), s.urlFor("/header.jpg")),
		HeaderStatic:   stringOrDefault(a.Header(), s.urlFor("/header.jpg")),
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
		Fields: algorithms.Map(a.Attributes(), func(a models.ActorAttribute) Field {
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
			Note:      a.Actor.Note(),
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
	Pinned             bool               `json:"pinned"`
	Bookmarked         bool               `json:"bookmarked"`
	Content            string             `json:"content"`
	Filtered           []any              `json:"filtered,omitempty"`
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
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Printf("panic in Status: %+v: %v\n", st, r)

	// 	}
	// }()
	createdAt := st.ObjectID.ToTime()
	return &Status{
		ID:                 st.ObjectID,
		CreatedAt:          createdAt.Round(time.Second).Format("2006-01-02T15:04:05.000Z"),
		EditedAt:           maybeEditedAt(createdAt, st.UpdatedAt),
		InReplyToID:        st.InReplyToID,
		InReplyToAccountID: st.InReplyToActorID,
		Sensitive:          st.Sensitive(),
		SpoilerText:        st.SpoilerText(),
		Visibility: func() models.Visibility {
			if st.Visibility == "limited" {
				return "private"
			}
			return st.Visibility
		}(),
		Language: func() any {
			// todo return nil if no language
			if st.Reblog != nil {
				return nil
			}
			if st.Language() == "" {
				return "en"
			}
			return st.Language()
		}(),
		URI: st.URI(),
		URL: func() any {
			if st.Reblog != nil {
				return nil
			}
			return st.URI()
		}(),
		RepliesCount:     st.RepliesCount,
		ReblogsCount:     st.ReblogsCount,
		FavouritesCount:  st.FavouritesCount,
		Favourited:       st.Reaction != nil && st.Reaction.Favourited,
		Reblogged:        st.Reaction != nil && st.Reaction.Reblogged,
		Muted:            st.Reaction != nil && st.Reaction.Muted,
		Bookmarked:       st.Reaction != nil && st.Reaction.Bookmarked,
		Content:          st.Note(),
		Reblog:           s.Status(st.Reblog),
		Account:          s.Account(st.Actor),
		MediaAttachments: s.MediaAttachments(st.Attachments()),
		// Mentions:         s.Mentions(st.Mentions),
		// Tags:             s.Tags(st.Tags),
		Emojis: nil,
		Card:   nil,
		// Poll:             s.Poll(st.Poll),
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
				ID:       a.ObjectID,
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
	Focus         *MetaFocus  `json:"focus,omitempty"`
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
	VotersCount *int         `json:"voters_count"`
	Voted       bool         `json:"voted"`
	OwnVotes    []int        `json:"own_votes"`
	Options     []PollOption `json:"options"`
	Emojis      []any        `json:"emojis"`
}

type PollOption struct {
	Title      string `json:"title"`
	VotesCount int    `json:"votes_count"`
}

func (s *Serialiser) Poll(p *models.StatusPoll) *Poll {
	if p == nil {
		return nil
	}
	poll := &Poll{
		ID:         p.StatusID,
		ExpiresAt:  p.ExpiresAt.Format("2006-01-02T15:04:05.006Z"),
		Expired:    p.ExpiresAt.Before(time.Now()),
		Multiple:   p.Multiple,
		VotesCount: p.VotesCount,
		Voted:      false,
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
	return poll
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
	Emojis           []any              `json:"emojis"`
}

func (s *Serialiser) StatusEdit(st *models.Status) *StatusEdit {
	return &StatusEdit{
		Content:     st.Note(),
		SpoilerText: st.SpoilerText(),
		Sensitive:   st.Sensitive(),
		CreatedAt: func() time.Time {
			createdAt := st.ObjectID.ToTime()
			if st.UpdatedAt.After(createdAt) {
				return st.UpdatedAt
			}
			return createdAt
		}().Format("2006-01-02T15:04:05.006Z"),
		Account: s.Account(st.Actor),
		// Poll:             s.Poll(st.Poll),
		// MediaAttachments: s.MediaAttachments(st.Attachments),
	}
}

const (
	PREVIEW_MAX_WIDTH  = 560
	PREVIEW_MAX_HEIGHT = 415
)

func (s *Serialiser) MediaAttachments(attachments []*models.Attachment) []*MediaAttachment {
	return algorithms.Map(
		attachments,
		s.mediaAttachment,
	)
}

func (s *Serialiser) mediaAttachment(att *models.Attachment) *MediaAttachment {
	return &MediaAttachment{
		// ID:         att.ID,
		Type:       att.ToType(),
		URL:        s.mediaOriginalURL(att),
		PreviewURL: s.mediaOriginalURL(att),
		// PreviewURL: s.mediaPreviewURL(att),
		RemoteURL: att.URL,
		Meta: Meta{
			Focus:    focus(att),
			Original: originalMetaFormat(att),
			Small:    smallMetaFormat(att),
		},
		Description: att.Name,
		Blurhash:    att.Blurhash,
	}
}

func focus(att *models.Attachment) *MetaFocus {
	if att.FocalPoint.X == 0 && att.FocalPoint.Y == 0 {
		return nil
	}
	return &MetaFocus{
		X: att.FocalPoint.X,
		Y: att.FocalPoint.Y,
	}
}

func originalMetaFormat(att *models.Attachment) *MetaFormat {
	if att.Width == 0 || att.Height == 0 {
		return nil
	}
	return &MetaFormat{
		Width:  att.Width,
		Height: att.Height,
		Size:   fmt.Sprintf("%dx%d", att.Width, att.Height),
		Aspect: float64(att.Width) / float64(att.Height),
	}
}

func smallMetaFormat(att *models.Attachment) *MetaFormat {
	if att.Width < PREVIEW_MAX_WIDTH && att.Height < PREVIEW_MAX_HEIGHT {
		return originalMetaFormat(att)
	}
	switch att.MediaType {
	case "image/jpeg", "image/gif", "image/png", "image/webp":
		h := att.Height
		w := att.Width

		if w > h {
			h = h * PREVIEW_MAX_WIDTH / w
			if h < 1 {
				h = 1
			}
			w = PREVIEW_MAX_WIDTH
		} else {
			w = w * PREVIEW_MAX_HEIGHT / h
			if w < 1 {
				w = 1
			}
			h = PREVIEW_MAX_HEIGHT
		}

		return &MetaFormat{
			Width:  w,
			Height: h,
			Size:   fmt.Sprintf("%dx%d", w, h),
			Aspect: float64(att.Width) / float64(att.Height),
		}
	default:
		// no preview available
		return nil
	}
}

func (s *Serialiser) mediaOriginalURL(att *models.Attachment) string {
	switch att.MediaType {
	// case "image/jpeg", "image/png", "image/gif", "image/webp":
	// 	// call through /media proxy to cache
	// 	return s.urlFor(fmt.Sprintf("/media/original/%d.%s", att.ID, att.Extension()))
	default:
		// otherwise return the remote URL
		return att.URL
	}
}

func (s *Serialiser) mediaPreviewURL(att *models.Attachment) string {
	if att.Width < PREVIEW_MAX_WIDTH && att.Height < PREVIEW_MAX_HEIGHT {
		return s.mediaOriginalURL(att)
	}
	// ext := att.Extension()
	switch att.MediaType {
	// case "image/png", "image/webp":
	// 	// request a JPEG preview
	// 	ext = "jpg"
	// 	fallthrough
	// case "image/jpeg", "image/gif":
	// 	// call through /media proxy to cache
	// 	return s.urlFor(fmt.Sprintf("/media/preview/%d.%s", att.ID, ext))
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

type FamiliarFollowers struct {
	ID       snowflake.ID `json:"id"`
	Accounts []*Account   `json:"accounts"`
}

type WebPushSubscription struct {
	ID        uint32 `json:"id"`
	Endpoint  string `json:"endpoint"`
	Alerts    Alerts `json:"alerts"`
	ServerKey string `json:"server_key"`
}

type Alerts struct {
	Mention       bool `json:"mention"`
	Status        bool `json:"status"`
	Reblog        bool `json:"reblog"`
	Follow        bool `json:"follow"`
	FollowRequest bool `json:"follow_request"`
	Favourite     bool `json:"favourite"`
	Poll          bool `json:"poll"`
	Update        bool `json:"update"`
}

func (s *Serialiser) WebPushSubscription(sub *models.PushSubscription) *WebPushSubscription {
	return &WebPushSubscription{
		ID:       sub.ID,
		Endpoint: sub.Endpoint,
		Alerts: Alerts{
			Follow:        sub.Follow,
			FollowRequest: sub.FollowRequest,
			Favourite:     sub.Favourite,
			Reblog:        sub.Reblog,
			Mention:       sub.Mention,
			Status:        sub.Status,
			Poll:          sub.Poll,
			Update:        sub.Update,
		},
		ServerKey: "BCk-QqERU0q-CfYZjcuB6lnyyOYfJ2AifKqfeGIm7Z-HiTU5T9eTG5GxVA0_OH5mMlI4UkkDTpaZwozy0TzdZ2M=",
	}
}
