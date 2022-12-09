package mastodon

import (
	"fmt"
	"strconv"

	"github.com/davecheney/m/m"
)

func serialize(a *m.Account) map[string]any {
	return map[string]any{
		"id":       strconv.Itoa(int(a.ID)),
		"username": a.Username,
		"acct": func(a *m.Account) string {
			if a.Local {
				return a.Username
			}
			return fmt.Sprintf("%s@%s", a.Username, a.Domain)
		}(a),
		"display_name":    a.DisplayName,
		"locked":          a.Locked,
		"bot":             a.Bot,
		"discoverable":    true,
		"group":           false, // todo
		"created_at":      a.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
		"note":            a.Note,
		"url":             a.URL(),
		"avatar":          stringOrDefault(a.Avatar, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"avatar_static":   stringOrDefault(a.Avatar, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"header":          stringOrDefault(a.Header, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"header_static":   stringOrDefault(a.Header, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"followers_count": a.FollowersCount,
		"following_count": a.FollowingCount,
		"statuses_count":  a.StatusesCount,
		"last_status_at":  a.LastStatusAt.Format("2006-01-02"),
		"noindex":         false, // todo
		"emojis":          []map[string]any{},
		"fields":          []map[string]any{},
	}
}
