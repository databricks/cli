package dbconnect

import (
	"fmt"
	"regexp"
	"strings"
)

// EnvKeyForServerless returns the environment key for a serverless version.
func EnvKeyForServerless(version string) string {
	// Strip leading 'v' or 'V' and lowercase
	normalized := strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")
	normalized = strings.ToLower(normalized)
	return fmt.Sprintf("serverless/serverless-v%s", normalized)
}

// EnvKeyForSparkVersion returns the environment key for a Spark version.
func EnvKeyForSparkVersion(sparkVersion string) string {
	return "dbr/" + sparkVersion
}

// PythonMinorFromRequires parses a PEP 440 requires-python string and extracts MAJOR.MINOR.
func PythonMinorFromRequires(requiresPython string) (string, error) {
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	match := re.FindStringSubmatch(requiresPython)
	if match == nil {
		return "", fmt.Errorf("cannot parse python version from %q", requiresPython)
	}
	return fmt.Sprintf("%s.%s", match[1], match[2]), nil
}
