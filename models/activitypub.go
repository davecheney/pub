package models

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
