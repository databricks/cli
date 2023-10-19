package build

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

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

var info Info

var once sync.Once

const DefaultSemver = "0.0.0-dev"

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

func initialize() {
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

	info = Info{
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

func GetInfo() Info {
	once.Do(initialize)
	return info
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
