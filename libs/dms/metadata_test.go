package dms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextVersion(t *testing.T) {
	for _, tt := range []struct {
		last string
		want int64
	}{
		{"", 1},
		{"4", 5},
	} {
		got, err := NextVersion(tt.last)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}

	_, err := NextVersion("not-a-number")
	assert.ErrorContains(t, err, "last_version_id")
}
