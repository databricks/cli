package build

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"
)

func TestGetDetails(t *testing.T) {
	GetInfo()
}

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want bool
	}{
		{"dev build with commit metadata", Info{Version: "1.12.0-dev+abc123"}, true},
		{"dev build without metadata", Info{Version: "1.12.0-dev"}, true},
		{"released version", Info{Version: "1.12.0"}, false},
		// A release candidate is not a dev build: it is built from a tag, so the
		// version constraints and update checks that dev builds bypass still apply.
		{"release candidate", Info{Version: "1.12.0-rc.1"}, false},
		// goreleaser marks snapshots explicitly, independent of the version string.
		{"snapshot of a release version", Info{Version: "1.12.0", IsSnapshot: true}, true},
		{"malformed version", Info{Version: "not-a-version"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.info.IsDevelopment())
		})
	}
}

// TestDefaultSemverSortsAboveLastRelease pins the invariant this version scheme
// exists for: a local build reports a version that sorts ABOVE the most recent
// release (it is built from main, so it is newer) and BELOW the release it will
// become. A bare "0.0.0-dev" sorted below every release instead.
func TestDefaultSemverSortsAboveLastRelease(t *testing.T) {
	v := "v" + DefaultSemver
	require.True(t, semver.IsValid(v), "DefaultSemver %q must be valid semver", DefaultSemver)
	require.Equal(t, devPrerelease, semver.Prerelease(v))

	// The release this dev build will become, e.g. v1.12.0 for 1.12.0-dev.
	next := strings.TrimSuffix(v, devPrerelease)
	assert.Positive(t, semver.Compare(next, v), "the upcoming release must sort above the dev build")
	assert.True(t, Info{Version: DefaultSemver}.IsDevelopment())
}

func TestGetSanitizedVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "version with plus",
			version:  "1.0.0+abc123",
			expected: "1.0.0-abc123",
		},
		{
			name:     "version with colon (Windows problematic)",
			version:  "1.0.0:dev",
			expected: "1.0.0-dev",
		},
		{
			name:     "version with forward slash (Windows problematic)",
			version:  "1.0.0/beta",
			expected: "1.0.0-beta",
		},
		{
			name:     "version with backslash (Windows problematic)",
			version:  "1.0.0\\test",
			expected: "1.0.0-test",
		},
		{
			name:     "version with multiple problematic characters",
			version:  "1.0.0+abc:123/test\\dev",
			expected: "1.0.0-abc-123-test-dev",
		},
		{
			name:     "clean version",
			version:  "1.0.0-dev",
			expected: "1.0.0-dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := Info{Version: tt.version}
			result := info.GetSanitizedVersion()
			assert.Equal(t, tt.expected, result)
		})
	}
}
