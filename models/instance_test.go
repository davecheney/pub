package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstances(t *testing.T) {
	db := setupTestDB(t)

	t.Run("create", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		instance := MockInstance(t, tx, "example.com")

		var i Instance
		err := tx.First(&i, "domain = ?", instance.Domain).Error
		require.NoError(err)
		require.Equal(instance.Domain, i.Domain)
	})

	t.Run("delete", func(t *testing.T) {
		require := require.New(t)
		tx := db.Begin()
		defer tx.Rollback()

		instance := MockInstance(t, tx, "example.com")
		err := tx.Delete(&instance).Error
		require.NoError(err)

		var i Instance
		err = tx.First(&i, "domain = ?", instance.Domain).Error
		require.Error(err)
		require.Equal("record not found", err.Error())
	})
}
