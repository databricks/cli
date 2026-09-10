package dms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextVersion(t *testing.T) {
	for _, tt := range []struct {
		last string
		want int
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

func TestResourceNames(t *testing.T) {
	// One format each, so a call only ever passes ids.
	assert.Equal(t, "deployments/dep-1", DeploymentName("dep-1"))
	assert.Equal(t, "deployments/dep-1/versions/2", VersionName("dep-1", 2))
}

func TestDeploymentIDFromName(t *testing.T) {
	id, err := DeploymentIDFromName("deployments/abc-123")
	require.NoError(t, err)
	assert.Equal(t, "abc-123", id)

	_, err = DeploymentIDFromName("abc-123")
	assert.Error(t, err)

	_, err = DeploymentIDFromName("deployments/")
	assert.Error(t, err)
}
