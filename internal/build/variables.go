package build

var (
	buildProjectName string = "cli"
	buildVersion     string = ""
)

var (
	buildBranch          string = "undefined"
	buildTag             string = "undefined"
	buildShortCommit     string = "00000000"
	buildFullCommit      string = "0000000000000000000000000000000000000000"
	buildCommitTimestamp string = "0"
	buildSummary         string = "v0.0.0"
)

var (
	buildMajor      string = "0"
	buildMinor      string = "0"
	buildPatch      string = "0"
	buildPrerelease string = ""
	buildIsSnapshot string = "false"
	buildTimestamp  string = "0"
)

// This function is used to set the build version for testing purposes.
func SetBuildVersion(version string) {
	buildVersion = version
	info.Version = version
}
