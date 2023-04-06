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

func MockActor(t *testing.T, tx *gorm.DB, name, domain string) *Actor {
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
