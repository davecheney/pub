package models

// AllTables returns a slice of all tables in the database.
func AllTables() []interface{} {
	return []interface{}{
		&ActivitypubRefresh{}, &ActivitypubOutboxRequest{},
		&Actor{}, &ActorAttribute{}, &ActorRefreshRequest{},
		&Account{}, &AccountList{}, &AccountListMember{}, &AccountRole{}, &AccountMarker{}, &AccountPreferences{},
		&Application{},
		&Conversation{},
		&Instance{}, &InstanceRule{},
		&Peer{},
		&PushSubscription{},
		&Reaction{}, &ReactionRequest{},
		&Relationship{}, &RelationshipRequest{},
		// &Notification{},
		&Status{}, &StatusPoll{}, &StatusPollOption{}, &StatusAttachment{}, &StatusMention{}, &StatusTag{},
		&StatusAttachmentRequest{},
		&Tag{},
		&Token{},
	}
}
