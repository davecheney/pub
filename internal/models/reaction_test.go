package models

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReactions(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Favourite", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		author := MockActor(t, tx, "alice", "example.com")
		favouritedBy := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, author, "This speech is my recital, I think it's very vital")

		reactions := NewReactions(tx)
		_, err := reactions.Favourite(status, favouritedBy)
		require.NoError(err)

		var reaction Reaction
		err = tx.Where("status_id = ? AND actor_id = ?", status.ID, favouritedBy.ID).First(&reaction).Error
		require.NoError(err)
		require.True(reaction.Favourited)

		var rr ReactionRequest
		err = tx.Where("actor_id = ? AND target_id = ?", favouritedBy.ID, status.ID).First(&rr).Error
		require.NoError(err)
		require.EqualValues("like", rr.Action)

		_, err = reactions.Unfavourite(status, favouritedBy)
		require.NoError(err)

		err = tx.Where("status_id = ? AND actor_id = ?", status.ID, favouritedBy.ID).First(&reaction).Error
		require.NoError(err)
		require.False(reaction.Favourited)

		err = tx.Where("actor_id = ? AND target_id = ?", favouritedBy.ID, status.ID).First(&rr).Error
		require.NoError(err)
		require.EqualValues("unlike", rr.Action)
	})

}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	require := require.New(t)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(err)

	err = db.AutoMigrate(AllTables()...)
	require.NoError(err)
	return db
}
