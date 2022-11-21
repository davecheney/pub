package mastodon

import "time"

type Status struct {
	// ID of the status in the database.
	Id string `json:"id"`
	// URI of the status for federation purposes.
	Uri string `json:"uri,omitempty"`
	// The date when this status was created.
	CreatedAt string `json:"created_at,omitempty"`
	// HTML-encoded status content.
	Content string `json:"content,omitempty"`
	//  Visibility of this status.
	Visibility string `json:"visibility,omitempty"`
}

type Account struct {
	// #  uri                           :string           default(""), not null
	// #  url                           :string
	// #  avatar_file_name              :string
	// #  avatar_content_type           :string
	// #  avatar_file_size              :integer
	// #  avatar_updated_at             :datetime
	// #  header_file_name              :string
	// #  header_content_type           :string
	// #  header_file_size              :integer
	// #  header_updated_at             :datetime
	// #  avatar_remote_url             :string
	// #  header_remote_url             :string           default(""), not null
	// #  last_webfingered_at           :datetime
	// #  inbox_url                     :string           default(""), not null
	// #  outbox_url                    :string           default(""), not null
	// #  shared_inbox_url              :string           default(""), not null
	// #  followers_url                 :string           default(""), not null
	// #  protocol                      :integer          default("ostatus"), not null
	// #  memorial                      :boolean          default(FALSE), not null
	// #  moved_to_account_id           :bigint(8)
	// #  featured_collection_url       :string
	// #  fields                        :jsonb
	// #  actor_type                    :string
	// #  discoverable                  :boolean
	// #  also_known_as                 :string           is an Array
	// #  silenced_at                   :datetime
	// #  suspended_at                  :datetime
	// #  hide_collections              :boolean
	// #  avatar_storage_schema_version :integer
	// #  header_storage_schema_version :integer
	// #  devices_url                   :string
	// #  suspension_origin             :integer
	// #  sensitized_at                 :datetime
	// #  trendable                     :boolean
	// #  reviewed_at                   :datetime
	// #  requested_review_at           :datetime
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	Domain      string    `json:"domain"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Note        string    `json:"note,omitempty"`
	DisplayName string    `json:"display_name"`
	Locked      bool      `json:"locked"`
}

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
