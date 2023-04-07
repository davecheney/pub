package models

import "time"

// activitypub support tables

// ActivitypubRefresh is a record of a request to refresh a URI.
type ActivitypubRefresh struct {
	ID uint32 `gorm:"primarykey;"`
	// CreatedAt is the time the request was created.
	CreatedAt time.Time
	// UpdatedAt is the time the request was last updated.
	UpdatedAt time.Time
	// URI is the URI to refresh.
	URI string `gorm:"size:255;not null;uniqueIndex"`
	// DependsOn is the URI that this URI depends on.
	// For example if URI is a reply, DependsOn is the URI of the status being replied to.
	// If URI is a status, DependsOn is the URI of the actor.
	DependsOn string `gorm:"size:255"`
	// Attempts is the number of times the request has been attempted.
	Attempts uint32 `gorm:"not null;default:0"`
	// LastAttempt is the time the request was last attempted.
	LastAttempt time.Time
	// LastResult is the result of the last attempt if it failed.
	LastResult string `gorm:"size:255"`
}
