package models

import (
	"fmt"
	"testing"

	"github.com/davecheney/pub/internal/crypto"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

	kp, err := crypto.GenerateRSAKeypair()
	require.NoError(err)

	actor := &Actor{
		ID:          snowflake.Now(),
		URI:         fmt.Sprintf("https://%s/%s", domain, name),
		Name:        name,
		Domain:      domain,
		DisplayName: name,
		Avatar:      "https://avatars.githubusercontent.com/u/1024?v=4",
		Header:      "https://avatars.githubusercontent.com/u/1024?v=4",
		PublicKey:   kp.PublicKey,
	}
	for _, opt := range opts {
		opt(actor)
	}
	require.NoError(tx.Create(actor).Error)
	return actor
}

func MockStatus(t *testing.T, tx *gorm.DB, actor *Actor, note string) *Status {
	t.Helper()
	require := require.New(t)

	status := &Status{
		ID:      snowflake.Now(),
		ActorID: actor.ID,
		Note:    note,
	}
	require.NoError(tx.Create(status).Error)
	return status
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
