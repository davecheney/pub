package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActors(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Refresh schedules a refresh", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		alice := MockActor(t, tx, "alice", "example.com")
		err := NewActors(tx).Refresh(alice)
		require.NoError(err)

		var req ActorRefreshRequest
		require.NoError(tx.First(&req, "actor_id = ?", alice.ID).Error)
		require.Equal(alice.ID, req.ActorID)

		// Refreshing again should not create a new request
		err = NewActors(tx).Refresh(alice)
		require.NoError(err)
		require.NoError(tx.First(&req, "actor_id = ?", alice.ID).Error)
		require.Equal(alice.ID, req.ActorID)
		// require one row
		var count int64
		require.NoError(tx.Model(&ActorRefreshRequest{ActorID: alice.ID}).Count(&count).Error)
		require.Equal(int64(1), count)
	})
}
