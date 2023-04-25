package mastodon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoolOrBit(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte("true"))
		require.NoError(err)
		require.True(bool(b))
	})
	t.Run("false", func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte("false"))
		require.NoError(err)
		require.False(bool(b))
	})
	t.Run("1", func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte("1"))
		require.NoError(err)
		require.True(bool(b))
	})
	t.Run("0", func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte("0"))
		require.NoError(err)
		require.False(bool(b))
	})
	t.Run("2", func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte("2")) // any number != 0 is true
		require.NoError(err)
		require.True(bool(b))
	})
	t.Run(`"true"`, func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte(`"true"`))
		require.NoError(err)
		require.True(bool(b))
	})
	t.Run(`"false"`, func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte(`"false"`))
		require.NoError(err)
		require.False(bool(b))
	})
	t.Run(":1", func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte(":1"))
		require.Error(err)
	})
	t.Run(`{"foo": "bar"}`, func(t *testing.T) {
		require := require.New(t)

		var b BoolOrBit
		err := b.UnmarshalJSON([]byte(`{"foo": "bar"}`))
		require.Error(err)
	})
}
