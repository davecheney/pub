package models

import (
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

// pagination support for the Mastodon API.

func PaginateActors(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()

		limit, _ := strconv.Atoi(q.Get("limit"))
		switch {
		case limit > 40:
			limit = 80
		case limit <= 0:
			limit = 20
		}
		db = db.Limit(limit)

		offset, _ := strconv.Atoi(q.Get("offset"))
		db = db.Offset(offset)

		switch q.Get("order") {
		case "new":
			db = db.Order("id desc")
		case "active":
			db = db.Order("last_status_at desc")
		}
		return db
	}
}

func PaginateConversation(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()

		limit, _ := strconv.Atoi(q.Get("limit"))
		switch {
		case limit > 40:
			limit = 40
		case limit <= 0:
			limit = 20
		}
		db = db.Limit(limit)

		sinceID, _ := strconv.Atoi(r.URL.Query().Get("since_id"))
		if sinceID > 0 {
			db = db.Where("statuses.id > ?", sinceID)
		}
		minID, _ := strconv.Atoi(r.URL.Query().Get("min_id"))
		if minID > 0 {
			db = db.Where("statuses.id > ?", minID)
		}
		maxID, _ := strconv.Atoi(r.URL.Query().Get("max_id"))
		if maxID > 0 {
			db = db.Where("statuses.id < ?", maxID)
		}
		return db
	}
}

func PaginateRelationship(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()

		limit, _ := strconv.Atoi(q.Get("limit"))
		switch {
		case limit > 40:
			limit = 40
		case limit <= 0:
			limit = 20
		}
		db = db.Limit(limit)

		sinceID, _ := strconv.Atoi(r.URL.Query().Get("since_id"))
		if sinceID > 0 {
			db = db.Where("relationships.target_id > ?", sinceID)
		}
		minID, _ := strconv.Atoi(r.URL.Query().Get("min_id"))
		if minID > 0 {
			db = db.Where("relationships.target_id > ?", minID)
		}
		maxID, _ := strconv.Atoi(r.URL.Query().Get("max_id"))
		if maxID > 0 {
			db = db.Where("relationships.target_id < ?", maxID)
		}
		return db.Order("relationships.target_id desc")
	}
}

func PaginateStatuses(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()

		limit, _ := strconv.Atoi(q.Get("limit"))
		switch {
		case limit > 40:
			limit = 40
		case limit <= 0:
			limit = 20
		}
		db = db.Limit(limit)

		// so there's a trick with min_id. If you pass a min_id, we need to find `limit` statuses that are above the min_id.
		// We can do this by sorting by the latest and counting back until we either hit limit, or hit the min_id, but that
		// creates a problem that if there are more that `limit` statuses between the min_id and the latest, we'll see the latest
		// `limit`, not the _earliest_ `limit`.
		//
		// Mastodon seems to handle this by outsourcing the problem to redis, 'natch. We can't do that, so when min_id is passed,
		// we'll sort ascending. This will wor, but it creates the problem that all clients *probably* expect statuses to be sorted
		// in descending order, which again Mastodon does for free. The simplest way to handle this is to sort the statuses in descending
		// order, during rendering.

		// mostly based on https://github.com/mastodon/mastodon/blob/main/app/models/feed.rb#L22

		maxID := q.Get("max_id")
		minID := q.Get("min_id")
		sinceID := q.Get("since_id")
		switch minID {
		case "":
			// no min_id provided, so we'll sort descending
			db = db.Order("statuses.id desc")
			if maxID != "" {
				db = db.Where("statuses.id < ?", maxID)
			}
			if sinceID != "" {
				db = db.Where("statuses.id > ?", sinceID)
			}
		default:
			// min_id provided, so we'll sort ascending
			// since_id is ignored
			db = db.Order("statuses.id asc")
			db = db.Where("statuses.id > ?", minID)
			if maxID != "" {
				db = db.Where("statuses.id < ?", maxID)
			}
		}
		return db
	}
}
