package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccounts(t *testing.T) {
	db := setupTestDB(t)

	t.Run("create", func(t *testing.T) {
		require := require.New(t)

		tx := db.Begin()
		defer tx.Rollback()

		instance := MockInstance(t, tx, "example.com")
		account, err := NewAccounts(tx).Create(instance, "alice", "alice@example.com", "password")
		require.NoError(err)
		require.NotNil(account)
	})

	t.Run("delete", func(t *testing.T) {
		require := require.New(t)

		tx := db.Begin()
		defer tx.Rollback()

		instance := MockInstance(t, tx, "example.com")
		account, err := NewAccounts(tx).Create(instance, "alice", "alice@example.com", "password")
		require.NoError(err)
		require.NotNil(account)

		err = tx.Delete(account).Error
		require.NoError(err)
	})

	t.Run("delete actors' account fails", func(t *testing.T) {
		require := require.New(t)

		tx := db.Begin()
		defer tx.Rollback()

		instance := MockInstance(t, tx, "example.com")
		account, err := NewAccounts(tx).Create(instance, "alice", "alice@example.com", "password")
		require.NoError(err)
		require.NotNil(account)

		err = tx.Delete(account.Actor).Error
		require.Error(err)
	})

	t.Run("delete account does not delete their actor", func(t *testing.T) {
		require := require.New(t)

		tx := db.Begin()
		defer tx.Rollback()

		instance := MockInstance(t, tx, "example.com")
		account, err := NewAccounts(tx).Create(instance, "alice", "alice@example.com", "password")
		require.NoError(err)
		require.NotNil(account)

		err = tx.Delete(account).Error
		require.NoError(err)

		var actor Actor
		err = tx.Where("id = ?", account.ActorID).First(&actor).Error
		require.NoError(err)
		require.NotNil(actor)
	})
}
