package build

import (
	"strconv"
	"sync"
	"time"
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

func initialize() {
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
