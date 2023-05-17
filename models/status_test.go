package models

// func TestStatus(t *testing.T) {
// 	db := setupTestDB(t)

// 	t.Run("Assert creating status creates unique conversation", func(t *testing.T) {
// 		require := require.New(t)
// 		tx := db.Begin()
// 		defer tx.Rollback()

// 		alice := MockActor(t, tx, "alice", "example.com")
// 		status := MockStatus(t, tx, alice, "Hello world")

// 		require.NotNil(status.Conversation)
// 		require.NotEmpty(status.Conversation.ID)

// 		var conv Conversation
// 		err := tx.First(&conv, status.ConversationID).Error
// 		require.NoError(err)
// 		require.Equal(status.ConversationID, conv.ID)
// 		require.EqualValues("public", conv.Visibility)
// 	})

// 	t.Run("Assert status can be deleted", func(t *testing.T) {
// 		require := require.New(t)
// 		tx := db.Begin()
// 		defer tx.Rollback()

// 		alice := MockActor(t, tx, "alice", "example.com")
// 		status := MockStatus(t, tx, alice, "Hello world")

// 		err := tx.Delete(status).Error
// 		require.NoError(err)
// 	})

// 	t.Run("Assert reblog creates a new status and conversation", func(t *testing.T) {
// 		require := require.New(t)
// 		tx := db.Begin()
// 		defer tx.Rollback()

// 		alice := MockActor(t, tx, "alice", "example.com")
// 		bob := MockActor(t, tx, "bob", "example.com")
// 		status := MockStatus(t, tx, alice, "Hello world")

// 		reblogged, err := NewReactions(tx).Reblog(status, bob)
// 		require.NoError(err)
// 		require.NotNil(reblogged)

// 		require.NotEqual(status.ID, reblogged.ID)
// 		require.NotEqual(status.ConversationID, reblogged.ConversationID)
// 	})

// 	t.Run("Assert status can be deleted after being rebloged", func(t *testing.T) {
// 		require := require.New(t)
// 		tx := db.Begin()
// 		defer tx.Rollback()

// 		alice := MockActor(t, tx, "alice", "example.com")
// 		bob := MockActor(t, tx, "bob", "example.com")
// 		status := MockStatus(t, tx, alice, "Hello world")

// 		reblogged, err := NewReactions(tx).Reblog(status, bob)
// 		require.NoError(err)
// 		require.NotNil(reblogged)

// 		err = tx.Delete(status).Error
// 		require.NoError(err)
// 	})

// 	t.Run("Assert status can be deleted after being favourited", func(t *testing.T) {
// 		require := require.New(t)
// 		tx := db.Begin()
// 		defer tx.Rollback()

// 		alice := MockActor(t, tx, "alice", "example.com")
// 		bob := MockActor(t, tx, "bob", "example.com")
// 		status := MockStatus(t, tx, alice, "Hello world")

// 		favourited, err := NewReactions(tx).Favourite(status, bob)
// 		require.NoError(err)
// 		require.NotNil(favourited)

// 		err = tx.Delete(status).Error
// 		require.NoError(err)
// 	})
// }

// func TestStatuses(t *testing.T) {
// 	db := setupTestDB(t)

// 	t.Run("FindOrCreate", func(t *testing.T) {
// 		t.Run("Assert status is created if it doesn't exist", func(t *testing.T) {
// 			require := require.New(t)
// 			tx := db.Begin()
// 			defer tx.Rollback()

// 			alice := MockActor(t, tx, "alice", "example.com")
// 			status, err := NewStatuses(tx).FindOrCreate("https://example.com/status/1", func(uri string) (*Status, error) {
// 				return &Status{
// 					ActorID: alice.ID,
// 					URI:     uri,
// 					Conversation: &Conversation{
// 						Visibility: "public",
// 					},
// 					Note: "Hello world",
// 				}, nil
// 			})
// 			require.NoError(err)
// 			require.NotNil(status)
// 			require.EqualValues("Hello world", status.Note)
// 			require.NotNil(status.Conversation)
// 			require.NotEmpty(status.Conversation.ID)
// 		})

// 		t.Run("Assert status is found if it exists", func(t *testing.T) {
// 			require := require.New(t)
// 			tx := db.Begin()
// 			defer tx.Rollback()

// 			alice := MockActor(t, tx, "alice", "example.com")
// 			st := MockStatus(t, tx, alice, "Hello world")
// 			status, err := NewStatuses(tx).FindOrCreate(st.URI, func(uri string) (*Status, error) {
// 				return nil, errors.New("should not be called")
// 			})
// 			require.NoError(err)
// 			require.NotNil(status)
// 			require.EqualValues("Hello world", status.Note)
// 			require.NotNil(status.Conversation)
// 			require.NotEmpty(status.Conversation.ID)
// 		})
// 	})

// 	t.Run("Create", func(t *testing.T) {
// 		t.Run("Create status without parent generates new conversation", func(t *testing.T) {
// 			require := require.New(t)
// 			tx := db.Begin()
// 			defer tx.Rollback()

// 			alice := MockActor(t, tx, "alice", "example.com")
// 			status, err := NewStatuses(tx).Create(
// 				alice,
// 				nil,
// 				"public",
// 				false,
// 				"",
// 				"en",
// 				"Hello world",
// 			)
// 			require.NoError(err)
// 			require.NotNil(status)
// 			require.EqualValues("Hello world", status.Note)
// 			require.NotNil(status.Conversation)
// 			require.EqualValues("public", status.Conversation.Visibility)

// 			var conv Conversation
// 			err = tx.First(&conv, status.ConversationID).Error
// 			require.NoError(err)
// 			require.Equal(status.ConversationID, conv.ID)
// 		})
// 		t.Run("Create status with parent uses parent conversation", func(t *testing.T) {
// 			require := require.New(t)
// 			tx := db.Begin()
// 			defer tx.Rollback()

// 			alice := MockActor(t, tx, "alice", "example.com")
// 			parent := MockStatus(t, tx, alice, "Hello world")
// 			status, err := NewStatuses(tx).Create(
// 				alice,
// 				parent,
// 				"public",
// 				false,
// 				"",
// 				"en",
// 				"Hello world to you too",
// 			)
// 			require.NoError(err)
// 			require.NotNil(status)
// 			require.EqualValues("Hello world to you too", status.Note)
// 			require.NotNil(status.Conversation)
// 			require.EqualValues("public", status.Conversation.Visibility)
// 			require.EqualValues(parent.ConversationID, status.ConversationID)

// 			// there should be only one conversation in the db
// 			var convs []Conversation
// 			err = tx.Find(&convs).Error
// 			require.NoError(err)
// 			require.Len(convs, 1)
// 		})
// 	})
// }
