package mastodon

import (
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

type Status struct {
	gorm.Model
	AccountID          uint
	Account            Account
	InReplyToID        *uint
	InReplyToAccountID *uint
	Sensitive          bool
	SpoilerText        string
	Visibility         string
	Language           string
	RepliesCount       int
	ReblogsCount       int
	FavouritesCount    int
	Content            string
}

func (s *Status) serialize() map[string]any {
	return map[string]any{
		"id":                     strconv.Itoa(int(s.ID)),
		"created_at":             s.CreatedAt.UTC().Format("2006-01-02T15:04:05.006Z"),
		"in_reply_to_id":         s.InReplyToID,
		"in_reply_to_account_id": s.InReplyToAccountID,
		"sensitive":              s.Sensitive,
		"spoiler_text":           s.SpoilerText,
		"visibility":             s.Visibility,
		"language":               s.Language,
		"uri":                    fmt.Sprintf("https://cheney.net/users/%s/statuses/%d", s.Account.Username, s.ID),
		"url":                    fmt.Sprintf("https://cheney.net/@%s/%d", s.Account.Username, s.ID),
		"replies_count":          s.RepliesCount,
		"reblogs_count":          s.ReblogsCount,
		"favourites_count":       s.FavouritesCount,
		"favourited":             false,
		"reblogged":              false,
		"muted":                  false,
		"bookmarked":             false,
		"content":                s.Content,
		"reblog":                 nil,
		"application":            nil,
		"account":                s.Account.serialize(),
		"media_attachments":      []map[string]any{},
		"mentions":               []map[string]any{},
		"tags":                   []map[string]any{},
		"emojis":                 []map[string]any{},
		"card":                   nil,
		"poll":                   nil,
	}
}
