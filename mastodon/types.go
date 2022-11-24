package mastodon

type Instance struct {
	// The domain name of the instance.
	URI string `json:"uri"`
	// The title of the website.
	Title string `json:"title"`
	// Admin-defined description of the Mastodon site.
	Description string `json:"description,omitempty"`
	// A shorter description defined by the admin.
	ShortDescription string `json:"short_description,omitempty"`
	// An email that may be contacted for any inquiries.
	Email string `json:"email"`
	//  The version of Mastodon installed on the instance.
	Version string `json:"version"`
	// Primary languages of the website and its staff.
	Languages []string `json:"languages"`
	// Whether registrations are enabled.
	Registrations bool `json:"registrations"`
	// Whether registrations require moderator approval.
	ApprovalRequired bool `json:"approval_required"`
	// Whether invites are enabled.
	InvitesEnabled bool `json:"invites_enabled"`
	// URLs of interest for clients apps.
	URLs map[string]any `json:"urls"`
	// Statistics about how much information the instance contains.
	Stats struct {
		// The number of users on the instance.
		UserCount int `json:"user_count"`
		// The number of statuses on the instance.
		StatusCount int `json:"status_count"`
		// The number of domains federated with the instance.
		DomainCount int `json:"domain_count"`
	} `json:"stats"`
}
