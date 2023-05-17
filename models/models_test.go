package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/davecheney/pub/internal/crypto"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// WithType sets the type of an actor.
func WithType(t ActorType) func(*Actor) {
	return func(a *Actor) {
		a.Type = t
	}
}

// MockActor creates a new actor in the database.
func MockActor(t *testing.T, tx *gorm.DB, name, domain string, opts ...func(*Actor)) *Actor {
	t.Helper()
	require := require.New(t)

	_, err := crypto.GenerateRSAKeypair()
	require.NoError(err)

	obj := &Object{
		Properties: map[string]any{
			"published":         time.Now().Format(time.RFC3339),
			"id":                fmt.Sprintf("https://%s/%s", domain, name),
			"type":              "Person",
			"preferredUsername": name,
			"displayName":       name,
		},
	}
	require.NoError(tx.Create(&obj).Error)
	var actor Actor
	require.NoError(tx.Scopes(PreloadActor).Where(&Actor{ObjectID: obj.ID}).Take(&actor).Error)
	return &actor
}

func MockStatus(t *testing.T, tx *gorm.DB, actor *Actor, note string) *Status {
	t.Helper()
	require := require.New(t)

	obj := &Object{
		Properties: map[string]any{
			"published":    time.Now().Format(time.RFC3339),
			"id":           fmt.Sprintf("https://%s/status/%d", actor.Domain, snowflake.Now()),
			"type":         "Note",
			"attributedTo": actor.URI(),
			"content":      note,
		},
	}
	require.NoError(tx.Create(&obj).Error)
	var status Status
	require.NoError(tx.Scopes(PreloadStatus).Where(&Status{ObjectID: obj.ID}).Take(&status).Error)
	return &status
}

func MockInstance(t *testing.T, tx *gorm.DB, domain string) *Instance {
	t.Helper()
	require := require.New(t)

	instance, err := NewInstances(tx).Create("example.com", "Example", "Example instance", "admin@example.com")
	require.NoError(err)
	return instance
}

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

	err = db.AutoMigrate(AllTables()...)
	require.NoError(err)

	// enable foreign key constraints
	err = db.Exec("PRAGMA foreign_keys = ON").Error
	require.NoError(err)

	return db
}
