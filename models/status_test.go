package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Assert creating status creates unique conversation", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		status := MockStatus(t, tx, alice, "Hello world")

		require.NotNil(status.Conversation)
		require.NotEmpty(status.Conversation.ID)

		var conv Conversation
		err := tx.First(&conv, status.ConversationID).Error
		require.NoError(err)
		require.Equal(status.ConversationID, conv.ID)
		require.EqualValues("public", conv.Visibility)
	})

	t.Run("Assert status can be deleted", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		status := MockStatus(t, tx, alice, "Hello world")

		err := tx.Delete(status).Error
		require.NoError(err)
	})

	t.Run("Assert reblog creates a new status and conversation", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		bob := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, alice, "Hello world")

		reblogged, err := NewReactions(tx).Reblog(status, bob)
		require.NoError(err)
		require.NotNil(reblogged)

		require.NotEqual(status.ID, reblogged.ID)
		require.NotEqual(status.ConversationID, reblogged.ConversationID)
	})

	t.Run("Assert status can be deleted after being rebloged", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		bob := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, alice, "Hello world")

		reblogged, err := NewReactions(tx).Reblog(status, bob)
		require.NoError(err)
		require.NotNil(reblogged)

		err = tx.Delete(status).Error
		require.NoError(err)
	})

	t.Run("Assert status can be deleted after being favourited", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		bob := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, alice, "Hello world")

		favourited, err := NewReactions(tx).Favourite(status, bob)
		require.NoError(err)
		require.NotNil(favourited)

		err = tx.Delete(status).Error
		require.NoError(err)
	})
}
