package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelationships(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Follow and Unfollow", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		bob := MockActor(t, tx, "bob", "example.com")

		// Follow
		_, err := NewRelationships(tx).Follow(alice, bob)
		require.NoError(err)

		forward, err := NewRelationships(tx).findOrCreate(alice, bob)
		require.NoError(err)
		require.Equal(true, forward.Following)
		require.Equal(false, forward.FollowedBy)

		err = tx.Find(alice).Error
		require.NoError(err)
		require.EqualValues(1, alice.FollowingCount)
		require.EqualValues(0, alice.FollowersCount)

		backward, err := NewRelationships(tx).findOrCreate(bob, alice)
		require.NoError(err)
		require.EqualValues(false, backward.Following)
		require.EqualValues(true, backward.FollowedBy)

		err = tx.Find(bob).Error
		require.NoError(err)
		require.EqualValues(1, bob.FollowersCount)
		require.EqualValues(0, bob.FollowingCount)

		// Unfollow
		_, err = NewRelationships(tx).Unfollow(alice, bob)
		require.NoError(err)

		forward, err = NewRelationships(tx).findOrCreate(alice, bob)
		require.NoError(err)
		require.Equal(false, forward.Following)
		require.Equal(false, forward.FollowedBy)

		err = tx.Find(alice).Error
		require.NoError(err)
		require.EqualValues(0, alice.FollowingCount)
		require.EqualValues(0, alice.FollowersCount)

		backward, err = NewRelationships(tx).findOrCreate(bob, alice)
		require.NoError(err)
		require.Equal(false, backward.Following)
		require.Equal(false, backward.FollowedBy)

		err = tx.Find(bob).Error
		require.NoError(err)
		require.EqualValues(0, bob.FollowersCount)
		require.EqualValues(0, bob.FollowingCount)
	})
}
