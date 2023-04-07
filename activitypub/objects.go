package activitypub

type Actor struct {
	Type      string `json:"type"`
	Inbox     string `json:"inbox"`
	Outbox    string `json:"outbox"`
	Endpoints struct {
		SharedInbox string `json:"sharedInbox"`
	} `json:"endpoints"`
}
