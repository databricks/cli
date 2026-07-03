package localenv

import (
	"fmt"
	"regexp"
	"strings"
)

var pythonVersionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

// lowerBoundClauseRe matches a single requires-python clause that establishes
// the Python floor to install: a lower-bound or pinning operator (>=, >, ==,
// ~=, ===) followed by a MAJOR.MINOR version. Upper-bound (<, <=) and exclusion
// (!=) clauses are deliberately not matched — they must not be chosen as the
// version to install.
var lowerBoundClauseRe = regexp.MustCompile(`(?:>=|===|==|~=|>)\s*(\d+)\.(\d+)`)

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

// PythonMinorFromRequires parses a PEP 440 requires-python string and extracts
// the MAJOR.MINOR of the Python version to install.
//
// A requires-python may hold several comma-separated clauses in any order
// (e.g. "<3.13,>=3.10"). The version to install is the lower bound, so a
// lower-bound / pinning clause (>=, >, ==, ~=, ===) is preferred; taking the
// first number in the string would pick an upper bound like "<3.13" — a
// version the specifier forbids. Only when no lower-bound clause is present do
// we fall back to the first MAJOR.MINOR found.
func PythonMinorFromRequires(requiresPython string) (string, error) {
	if m := lowerBoundClauseRe.FindStringSubmatch(requiresPython); m != nil {
		return fmt.Sprintf("%s.%s", m[1], m[2]), nil
	}
	match := pythonVersionRe.FindStringSubmatch(requiresPython)
	if match == nil {
		return "", fmt.Errorf("cannot parse python version from %q", requiresPython)
	}
	return fmt.Sprintf("%s.%s", match[1], match[2]), nil
}
