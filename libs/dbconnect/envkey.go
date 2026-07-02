package dbconnect

import (
	"fmt"
	"regexp"
	"strings"
)

var pythonVersionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

// NormalizeServerless returns the canonical "vN" spelling of a serverless
// version accepting "4", "v4", or "V4".
func NormalizeServerless(version string) string {
	return "v" + strings.TrimPrefix(strings.ToLower(version), "v")
}

// EnvKeyForServerless returns the environment key for a serverless version.
func EnvKeyForServerless(version string) string {
	return "serverless/serverless-" + NormalizeServerless(version)
}

// EnvKeyForSparkVersion returns the environment key for a Spark version.
func EnvKeyForSparkVersion(sparkVersion string) string {
	return "dbr/" + sparkVersion
}

// PythonMinorFromRequires parses a PEP 440 requires-python string and extracts MAJOR.MINOR.
func PythonMinorFromRequires(requiresPython string) (string, error) {
	match := pythonVersionRe.FindStringSubmatch(requiresPython)
	if match == nil {
		return "", fmt.Errorf("cannot parse python version from %q", requiresPython)
	}
	return fmt.Sprintf("%s.%s", match[1], match[2]), nil
}
