package algorithms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		require := require.New(t)

		var s []int
		got := Map(s, func(i int) int { return i })
		require.Equal(got, []int{})
	})
	t.Run("non-empty slice", func(t *testing.T) {
		require := require.New(t)

		s := []int{1, 2, 3}
		got := Map(s, func(i int) int { return i * 2 })
		require.Equal(got, []int{2, 4, 6})
	})
}
func TestFilter(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		require := require.New(t)

		var s []int
		got := Filter(s, func(i int) bool { return i%2 == 0 })
		require.Equal(got, []int{})
	})
	t.Run("non-empty slice", func(t *testing.T) {
		require := require.New(t)

		s := []int{1, 2, 3}
		got := Filter(s, func(i int) bool { return i%2 == 0 })
		require.Equal(got, []int{2})
	})
}
