package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReactions(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Favourite and Unfavourite", func(t *testing.T) {
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
		err = tx.Where("status_id = ? AND actor_id = ?", status.ObjectID, favouritedBy.ObjectID).First(&reaction).Error
		require.NoError(err)
		require.True(reaction.Favourited)

		var rr ReactionRequest
		err = tx.Where("actor_id = ? AND target_id = ?", favouritedBy.ObjectID, status.ObjectID).First(&rr).Error
		require.NoError(err)
		require.EqualValues("like", rr.Action)

		var st Status
		err = tx.Where(&Status{ObjectID: status.ObjectID}).First(&st).Error
		require.NoError(err)
		require.EqualValues(1, st.FavouritesCount)

		_, err = reactions.Unfavourite(status, favouritedBy)
		require.NoError(err)

		err = tx.Where("status_id = ? AND actor_id = ?", status.ObjectID, favouritedBy.ObjectID).First(&reaction).Error
		require.NoError(err)
		require.False(reaction.Favourited)

		err = tx.Where("actor_id = ? AND target_id = ?", favouritedBy.ObjectID, status.ObjectID).First(&rr).Error
		require.NoError(err)
		require.EqualValues("unlike", rr.Action)

		err = tx.Where(&Status{ObjectID: status.ObjectID}).First(&st).Error
		require.NoError(err)
		require.EqualValues(0, st.FavouritesCount)
	})

	t.Run("Pin and Unpin", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		author := MockActor(t, tx, "alice", "example.com")
		pinnedBy := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, author, "This speech is my recital, I think it's very vital")

		reactions := NewReactions(tx)
		_, err := reactions.Pin(status, pinnedBy)
		require.NoError(err)

		var reaction Reaction
		err = tx.Where("status_id = ? AND actor_id = ?", status.ObjectID, pinnedBy.ObjectID).First(&reaction).Error
		require.NoError(err)
		require.True(reaction.Pinned)

		_, err = reactions.Unpin(status, pinnedBy)
		require.NoError(err)

		err = tx.Where("status_id = ? AND actor_id = ?", status.ObjectID, pinnedBy.ObjectID).First(&reaction).Error
		require.NoError(err)
		require.False(reaction.Pinned)
	})

	t.Run("Reblog and Unreblog", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		author := MockActor(t, tx, "alice", "example.com")
		rebloggedBy := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, author, "This speech is my recital, I think it's very vital")

		reactions := NewReactions(tx)
		_, err := reactions.Reblog(status, rebloggedBy)
		require.NoError(err)

		var reaction Reaction
		err = tx.Where("status_id = ? AND actor_id = ?", status.ObjectID, rebloggedBy.ObjectID).First(&reaction).Error
		require.NoError(err)
		require.True(reaction.Reblogged)

		st, err := NewStatuses(tx).FindByID(status.ObjectID)
		require.NoError(err)
		require.EqualValues(1, st.ReblogsCount)

		_, err = reactions.Unreblog(status, rebloggedBy)
		require.NoError(err)

		err = tx.Where("status_id = ? AND actor_id = ?", status.ObjectID, rebloggedBy.ObjectID).First(&reaction).Error
		require.NoError(err)
		require.False(reaction.Reblogged)

		st, err = NewStatuses(tx).FindByID(status.ObjectID)
		require.NoError(err)
		require.EqualValues(0, st.ReblogsCount)
	})

	t.Run("Bookmark and Unbookmark", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		author := MockActor(t, tx, "alice", "example.com")
		bookmarkedBy := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, author, "This speech is my recital, I think it's very vital")

		reactions := NewReactions(tx)
		_, err := reactions.Bookmark(status, bookmarkedBy)
		require.NoError(err)

		var reaction Reaction
		err = tx.Where("status_id = ? AND actor_id = ?", status.ObjectID, bookmarkedBy.ObjectID).First(&reaction).Error
		require.NoError(err)
		require.True(reaction.Bookmarked)

		_, err = reactions.Unbookmark(status, bookmarkedBy)
		require.NoError(err)

		err = tx.Where("status_id = ? AND actor_id = ?", status.ObjectID, bookmarkedBy.ObjectID).First(&reaction).Error
		require.NoError(err)
		require.False(reaction.Bookmarked)
	})

	t.Run("delete an actor deletes their reactions", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		bob := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, alice, "This speech is my recital, I think it's very vital")

		reactions := NewReactions(tx)
		_, err := reactions.Favourite(status, bob)
		require.NoError(err)

		err = tx.Delete(bob).Error
		require.NoError(err)
	})

	t.Run("delete a status deletes its reactions", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		bob := MockActor(t, tx, "bob", "example.com")
		status := MockStatus(t, tx, alice, "This speech is my recital, I think it's very vital")

		reactions := NewReactions(tx)
		_, err := reactions.Favourite(status, bob)
		require.NoError(err)

		err = tx.Delete(status).Error
		require.NoError(err)
	})
}
