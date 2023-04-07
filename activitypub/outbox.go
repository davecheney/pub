package activitypub

import (
	"fmt"
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
)

func Outbox(env *Env, w http.ResponseWriter, r *http.Request) error {
	switch parseBool(r, "page") {
	case true:
		return outboxShow(env, w, r)
	default:
		return outboxIndex(env, w, r)
	}
}

func outboxIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	var count int64
	query := env.DB.Joins("JOIN actors ON actors.id = statuses.actor_id and actors.name = ? and actors.domain = ?", chi.URLParam(r, "name"), r.Host)
	if err := query.Model(&models.Status{}).Count(&count).Error; err != nil {
		return err
	}
	return to.JSON(w, map[string]any{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"id":         fmt.Sprintf("https://%s%s", r.Host, r.URL.Path),
		"type":       "OrderedCollection",
		"totalItems": count,
		"first":      fmt.Sprintf("https://%s%s?page=true", r.Host, r.URL.Path),
		"last":       fmt.Sprintf("https://%s%s?min_id=0&page=true", r.Host, r.URL.Path),
	})
}

func outboxShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	resp := map[string]any{
		"@context": []any{
			"https://www.w3.org/ns/activitystreams",
			map[string]any{
				"ostatus":          "http://ostatus.org#",
				"atomUri":          "ostatus:atomUri",
				"inReplyToAtomUri": "ostatus:inReplyToAtomUri",
				"conversation":     "ostatus:conversation",
				"sensitive":        "as:sensitive",
				"toot":             "http://joinmastodon.org/ns#",
				"votersCount":      "toot:votersCount",
				"blurhash":         "toot:blurhash",
				"focalPoint": map[string]any{
					"@container": "@list",
					"@id":        "toot:focalPoint",
				},
			},
		},
		"id":     r.URL.String(),
		"type":   "OrderedCollectionPage",
		"partOf": fmt.Sprintf("https://%s%s", r.Host, r.URL.Path),
	}
	var statuses []*models.Status
	query := env.DB.Joins("JOIN actors ON actors.id = statuses.actor_id and actors.name = ? and actors.domain = ?", chi.URLParam(r, "name"), r.Host)
	query = query.Scopes(models.PaginateStatuses(r), models.PreloadStatus).Preload("Actor")
	if err := query.Find(&statuses).Error; err != nil {
		return err
	}
	if len(statuses) > 0 {
		resp["next"] = fmt.Sprintf("https://%s%s?max_id=%d&page=true", r.Host, r.URL.Path, statuses[0].ID)
		resp["prev"] = fmt.Sprintf("https://%s%s?min_id=%d&page=true", r.Host, r.URL.Path, statuses[len(statuses)-1].ID)
	}
	resp["orderedItems"] = algorithms.Map(statuses, statusToItem)
	return to.JSON(w, resp)
}

func statusToItem(s *models.Status) *Item {
	return &Item{
		ID:        s.URI,
		Type:      statusType(s),
		Actor:     s.Actor.URI,
		Published: s.ID.ToTime().Format("2006-01-02T15:04:05Z"),
		To:        statusTo(s),
		CC:        statusCC(s),
		Object:    statusToObject(s),
	}
}

type Item struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Actor     string   `json:"actor"`
	Published string   `json:"published"`
	To        []string `json:"to"`
	CC        []string `json:"cc"`
	Object    any      `json:"object"`
}

func statusType(s *models.Status) string {
	if s.ReblogID != nil {
		return "Announce"
	}
	return "Create"
}

func statusTo(s *models.Status) []string {
	if s.Visibility == "public" {
		return []string{"https://www.w3.org/ns/activitystreams#Public"}
	}
	return []string{s.Actor.URI}
}

func statusCC(s *models.Status) []string {
	if s.ReblogID != nil {
		return []string{
			s.Reblog.Actor.URI,
			s.Actor.URI + "/followers",
		}
	}
	return []string{}
}

func statusToObject(s *models.Status) any {
	if s.ReblogID != nil {
		return s.Reblog.URI
	}
	return nil
}
