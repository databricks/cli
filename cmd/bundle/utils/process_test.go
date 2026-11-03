package utils

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		current string
		want    bool
	}{
		{"state newer major", "1.0.0", "0.300.0", true},
		{"state newer minor", "0.301.0", "0.300.0", true},
		{"state newer patch", "0.300.1", "0.300.0", true},
		{"same version", "0.300.0", "0.300.0", false},
		{"state older", "0.299.0", "0.300.0", false},
		// A dev build is built from main, so its version is the next release with a
		// -dev prerelease: newer than the last release, older than the release it
		// will become. Deploying with a dev build after a state written by the last
		// release is the normal case for a CLI developer and must not warn.
		{"state from last release, dev current", "0.300.0", "0.301.0-dev+abc123", false},
		// A released CLI reading a state written by a dev build of the same upcoming
		// release does warn: that build may have written fields this CLI lacks.
		{"dev state, released current", "0.301.0-dev+abc123", "0.300.0", true},
		// A prerelease sorts below its own release per semver.
		{"prerelease below release", "0.300.0-rc1", "0.300.0", false},
		{"release above prerelease", "0.300.0", "0.300.0-rc1", true},
		// Missing or malformed data must never produce a warning.
		{"empty state version", "", "0.300.0", false},
		{"empty current version", "0.300.0", "", false},
		{"malformed state version", "not-a-version", "0.300.0", false},
		{"malformed current version", "0.300.0", "not-a-version", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isNewerVersion(tt.state, tt.current))
		})
	}
}

func TestParseLastVersionID(t *testing.T) {
	tests := []struct {
		name       string
		deployment *bundledeployments.Deployment
		want       int
		wantErr    bool
	}{
		{"nil deployment", nil, 0, false},
		{"empty version", &bundledeployments.Deployment{}, 0, false},
		{"version zero", &bundledeployments.Deployment{LastVersionId: "0"}, 0, false},
		{"valid version", &bundledeployments.Deployment{LastVersionId: "7"}, 7, false},
		{"invalid version", &bundledeployments.Deployment{LastVersionId: "abc"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLastVersionID(tt.deployment)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
