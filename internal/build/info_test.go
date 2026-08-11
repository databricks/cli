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

func TestIsDevelopmentVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.12.0-dev+abc123", true},
		{"1.12.0-dev", true},
		// Accepted with or without a leading "v", since callers hold versions in
		// both forms (the schema's minimum version is v-prefixed).
		{"v1.12.0-dev", true},
		// A dev build on a prerelease release track; see devVersion.
		{"1.13.0-rc.2.dev", true},
		{"1.12.0", false},
		{"v1.12.0", false},
		{"1.12.0-rc.1", false},
		// "rc.2-dev" is a single identifier that merely ends in "dev", not a dev
		// marker. This is what naively appending "-dev" to a prerelease produced.
		{"1.13.0-rc.2-dev", false},
		{"not-a-version", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, IsDevelopmentVersion(tt.version))
		})
	}
}

// TestVersionOrdering pins the full ordering model this version scheme exists
// for. A dev build is built from main, so it must sort ABOVE the release it
// followed and BELOW the release it will become. The old "0.0.0-dev" scheme
// violated this: it sorted below every release, including bare "0.0.0".
//
// The list is in strictly ascending order; the test asserts every pair, so it
// covers both the neighbouring steps and the transitive relationships.
func TestVersionOrdering(t *testing.T) {
	ascending := []string{
		"0.0.0-dev",
		"0.0.0",
		"1.11.0-dev",
		"1.11.0",
		// Build metadata is not part of precedence, so this is EQUAL to the
		// bare 1.12.0-dev below and must sit between 1.11.0 and 1.12.0.
		"1.12.0-dev+abc123",
		"1.12.0-rc.1",
		// A dev build off a prerelease sorts after the rc it follows and before
		// the final release; see devVersion for why this shape exists.
		"1.12.0-rc.1.dev",
		"1.12.0-rc.2",
		"1.12.0",
		"1.12.1-dev",
		"1.12.1",
		"2.0.0-dev",
		"2.0.0",
	}

	// Build metadata is ignored for precedence, so this pair compares equal.
	require.Zero(t, semver.Compare("v1.12.0-dev+abc123", "v1.12.0-dev"))

	for i, lower := range ascending {
		require.True(t, semver.IsValid("v"+lower), "%q must be valid semver", lower)
		assert.Zero(t, semver.Compare("v"+lower, "v"+lower), "%q must equal itself", lower)

		for _, higher := range ascending[i+1:] {
			// 1.12.0-dev+abc123 and 1.12.0-rc.1 are adjacent in the list but
			// -dev sorts below -rc alphabetically, so they are still ordered.
			assert.Negative(t, semver.Compare("v"+lower, "v"+higher), "%q must sort below %q", lower, higher)
			assert.Positive(t, semver.Compare("v"+higher, "v"+lower), "%q must sort above %q", higher, lower)
		}
	}
}

// TestDevVersion covers both shapes .nextchanges/version can take. The
// prerelease case is for completeness with next_release_version() in
// internal/genkit/tagging.py, which bumps a prerelease when the file carries
// one; this repo has never published a prerelease.
func TestDevVersion(t *testing.T) {
	tests := []struct {
		next string
		want string
		// finalRelease is the release the dev build must sort below: the version
		// itself on a stable track, or the release the prerelease leads up to.
		finalRelease string
	}{
		{"1.12.0", "1.12.0-dev", "1.12.0"},
		{"2.0.0", "2.0.0-dev", "2.0.0"},
		// Appending "-dev" here would yield the prerelease "rc.2-dev", which is
		// not a dev marker, so IsDevelopmentVersion would report false and every
		// dev-build exemption would silently switch off.
		{"1.13.0-rc.2", "1.13.0-rc.2.dev", "1.13.0"},
	}

	for _, tt := range tests {
		t.Run(tt.next, func(t *testing.T) {
			got := devVersion(tt.next)
			assert.Equal(t, tt.want, got)
			require.True(t, semver.IsValid("v"+got), "%q must be valid semver", got)
			assert.True(t, IsDevelopmentVersion(got), "%q must be recognized as a dev build", got)
			// The dev build sits below the release it will become, and above the
			// prerelease it already contains, if any.
			assert.Negative(t, semver.Compare("v"+got, "v"+tt.finalRelease), "%q must sort below %q", got, tt.finalRelease)
			if tt.next != tt.finalRelease {
				assert.Positive(t, semver.Compare("v"+got, "v"+tt.next), "%q must sort above %q", got, tt.next)
			}
		})
	}
}

// TestDefaultSemverSortsAboveLastRelease applies the ordering above to the
// version an actual local build reports, which TestVersionOrdering cannot do
// because DefaultSemver tracks .nextchanges/version.
func TestDefaultSemverSortsAboveLastRelease(t *testing.T) {
	v := "v" + DefaultSemver
	require.True(t, semver.IsValid(v), "DefaultSemver %q must be valid semver", DefaultSemver)
	require.True(t, IsDevelopmentVersion(DefaultSemver))
	assert.True(t, Info{Version: DefaultSemver}.IsDevelopment())

	// The release this dev build will become, e.g. v1.12.0 for 1.12.0-dev.
	next := "v" + strings.TrimSuffix(strings.TrimSuffix(DefaultSemver, "-"+devIdentifier), "."+devIdentifier)
	require.NotEqual(t, v, next, "DefaultSemver must end with the dev identifier")
	assert.Positive(t, semver.Compare(next, v), "the upcoming release must sort above the dev build")
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
