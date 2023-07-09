package oauth

import (
	"testing"

	"github.com/davecheney/pub/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	require := require.New(t)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		TranslateError: true,
		Logger: logger.Default.LogMode(func() logger.LogLevel {
			return logger.Warn
		}()),
	})
	require.NoError(err)

	err = db.AutoMigrate(models.AllTables()...)
	require.NoError(err)

	// enable foreign key constraints
	err = db.Exec("PRAGMA foreign_keys = ON").Error
	require.NoError(err)

	return db
}
