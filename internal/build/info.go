package build

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	nextchanges "github.com/databricks/cli/.nextchanges"
	"golang.org/x/mod/semver"
)

type Info struct {
	ProjectName string
	Version     string

	Branch      string
	Tag         string
	ShortCommit string
	FullCommit  string
	CommitTime  time.Time
	Summary     string

	Major      int64
	Minor      int64
	Patch      int64
	Prerelease string
	IsSnapshot bool
	BuildTime  time.Time
}

// GetSanitizedVersion removes characters from version string that might be problematic in file paths.
// Particularly important for Windows which has restrictions on certain characters.
func (i Info) GetSanitizedVersion() string {
	// Replace + with - (used in version metadata like "1.0.0+abc123")
	version := strings.ReplaceAll(i.Version, "+", "-")
	// Remove any other potentially problematic characters
	version = strings.ReplaceAll(version, ":", "-")
	version = strings.ReplaceAll(version, "/", "-")
	version = strings.ReplaceAll(version, "\\", "-")
	return version
}

// devPrerelease marks a build that was not produced from a release tag.
const devPrerelease = "-dev"

// DefaultSemver is the version reported when buildVersion was not injected,
// i.e. a plain "go build" rather than a goreleaser build. It is the next release
// version with a -dev prerelease, so it sorts above the latest release and below
// the release it will become:
//
//	Compare(v1.11.0, v1.12.0-dev+sha) = -1
//	Compare(v1.12.0, v1.12.0-dev+sha) = +1
//
// A bare "0.0.0-dev" would instead sort below every published release, even
// though a local build is built from main and is therefore newer than the
// latest release. This matches what goreleaser produces for snapshot builds
// (see snapshot.version_template in .goreleaser.yaml).
var DefaultSemver = nextchanges.Version + devPrerelease

// IsDevelopmentVersion reports whether a version string is a development build's.
// It keys off the -dev prerelease rather than a specific version number, so it
// keeps working as the next release version changes. The version may be given
// with or without a leading "v".
//
// Prefer Info.IsDevelopment when you have the running build's Info; this is for
// callers holding only a version string (e.g. one read from a file or a
// parameter), so the definition of "development build" lives in one place.
func IsDevelopmentVersion(version string) bool {
	return semver.Prerelease("v"+strings.TrimPrefix(version, "v")) == devPrerelease
}

// IsDevelopment reports whether this binary was built from a development or
// snapshot build rather than a release tag.
func (i Info) IsDevelopment() bool {
	return i.IsSnapshot || IsDevelopmentVersion(i.Version)
}

// getDefaultBuildVersion uses build information stored by Go itself
// to synthesize a build version if one wasn't set.
// This is necessary if the binary was not built through goreleaser.
func getDefaultBuildVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		panic("unable to read build info")
	}

	m := make(map[string]string)
	for _, s := range bi.Settings {
		m[s.Key] = s.Value
	}

	out := DefaultSemver

	// Append revision as build metadata.
	if v, ok := m["vcs.revision"]; ok {
		// First 12 characters of the commit SHA is plenty to identify one.
		out = fmt.Sprintf("%s+%s", out, v[0:12])
	}

	return out
}

func initialize() Info {
	// If buildVersion is empty it means the binary was NOT built through goreleaser.
	// We try to pull version information from debug.BuildInfo().
	if buildVersion == "" {
		buildVersion = getDefaultBuildVersion()
	}

	// Confirm that buildVersion is valid semver.
	// Note that the semver package requires a leading 'v'.
	if !semver.IsValid("v" + buildVersion) {
		panic(fmt.Sprintf(`version is not a valid semver string: "%s"`, buildVersion))
	}

	return Info{
		ProjectName: buildProjectName,
		Version:     buildVersion,

		Branch:      buildBranch,
		Tag:         buildTag,
		ShortCommit: buildShortCommit,
		FullCommit:  buildFullCommit,
		CommitTime:  parseTime(buildCommitTimestamp),
		Summary:     buildSummary,

		Major:      parseInt(buildMajor),
		Minor:      parseInt(buildMinor),
		Patch:      parseInt(buildPatch),
		Prerelease: buildPrerelease,
		IsSnapshot: parseBool(buildIsSnapshot),
		BuildTime:  parseTime(buildTimestamp),
	}
}

var getInfo = sync.OnceValue(initialize)

func GetInfo() Info {
	return getInfo()
}

func parseInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		panic(err)
	}
	return b
}

func parseTime(s string) time.Time {
	return time.Unix(parseInt(s), 0)
}
