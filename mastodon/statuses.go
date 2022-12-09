package mastodon

import (
	"fmt"
	"net/url"
	"path"

	"github.com/davecheney/m/internal/snowflake"
	"github.com/davecheney/m/m"
)

func serializeStatus(s *m.Status) map[string]any {
	return map[string]any{
		"id":                     toString(s.ID),
		"created_at":             snowflake.IDToTime(s.ID).UTC().Format("2006-01-02T15:04:05.006Z"),
		"in_reply_to_id":         stringOrNull(s.InReplyToID),
		"in_reply_to_account_id": stringOrNull(s.InReplyToAccountID),
		"sensitive":              s.Sensitive,
		"spoiler_text":           s.SpoilerText,
		"visibility":             s.Visibility,
		"language":               "en", // s.Language,
		"uri":                    s.URI,
		"url": func(s *m.Status) string {
			u, err := url.Parse(s.URI)
			if err != nil {
				return ""
			}
			id := path.Base(u.Path)
			return fmt.Sprintf("%s://%s/@%s/%s", u.Scheme, s.Account.Domain, s.Account.Username, id)
		}(s),
		"replies_count":    s.RepliesCount,
		"reblogs_count":    s.ReblogsCount,
		"favourites_count": s.FavouritesCount,
		// "favourited":             false,
		// "reblogged":              false,
		// "muted":                  false,
		// "bookmarked":             false,
		"content":           s.Content,
		"text":              nil,
		"reblog":            nil,
		"application":       nil,
		"account":           serialize(s.Account),
		"media_attachments": []map[string]any{},
		"mentions":          []map[string]any{},
		"tags":              []map[string]any{},
		"emojis":            []map[string]any{},
		"card":              nil,
		"poll":              nil,
	}
}
