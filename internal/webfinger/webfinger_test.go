package webfinger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcctParse(t *testing.T) {
	tc := []struct {
		in     string
		expect Acct
	}{
		{"acct:foo@bar.com", Acct{User: "foo", Host: "bar.com"}},
	}
	for _, tt := range tc {
		t.Run(tt.in, func(t *testing.T) {
			req := require.New(t)
			got, err := Parse(tt.in)
			req.NoError(err)
			req.Equal(tt.expect, *got)
			req.Equal(tt.in, got.String())
		})
	}
}
