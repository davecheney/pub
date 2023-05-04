package models

import "github.com/davecheney/pub/internal/snowflake"

// activitypub support tables

// ActivitypubRefresh is a record of a request to refresh a URI.
type ActivitypubRefresh struct {
	Request

	// URI is the URI to refresh.
	URI string `gorm:"size:255;not null;uniqueIndex"`
	// DependsOn is the URI that this URI depends on.
	// For example if URI is a reply, DependsOn is the URI of the status being replied to.
	// If URI is a status, DependsOn is the URI of the actor.
	DependsOn string `gorm:"size:255"`
}

// ActivitypubOutboxRequest is a record of a request to send a status to an actor on a remote server.
type ActivitypubOutboxRequest struct {
	Request

	// StatusID is the ID of the status to send.
	StatusID snowflake.ID `gorm:"not null"`
	Status   *Status      `gorm:"constraint:OnDelete:CASCADE;<-:false;"`

	// ActorID is the ID of the remote actor to send the status to.
	ActorID snowflake.ID `gorm:"not null"`
	Actor   *Actor       `gorm:"constraint:OnDelete:CASCADE;<-:false;"`
}
