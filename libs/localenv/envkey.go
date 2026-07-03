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

// boundedClauseRe matches a MAJOR.MINOR that is governed by an upper-bound or
// exclusion operator (<, <=, !=). Such a version is forbidden or capped, never a
// version to install, so a spec whose only version is bounded this way has no
// usable floor.
var boundedClauseRe = regexp.MustCompile(`(?:<=|<|!=)\s*(\d+)\.(\d+)`)

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
// lower-bound / pinning clause (>=, >, ==, ~=, ===) is preferred. A bare
// MAJOR.MINOR with no operator (e.g. "3.12") is also accepted as the floor.
// But a spec whose only version is governed by an upper-bound/exclusion
// operator (e.g. "<3.13" or "!=3.12") has no usable floor: picking that number
// would select a version the specifier forbids, so we error instead of guessing.
func PythonMinorFromRequires(requiresPython string) (string, error) {
	if m := lowerBoundClauseRe.FindStringSubmatch(requiresPython); m != nil {
		return fmt.Sprintf("%s.%s", m[1], m[2]), nil
	}
	match := pythonVersionRe.FindStringSubmatch(requiresPython)
	if match == nil {
		return "", fmt.Errorf("cannot parse python version from %q", requiresPython)
	}
	// A version exists but not behind a lower-bound/pin operator. If it is behind
	// an upper-bound/exclusion operator, there is no floor to install.
	if boundedClauseRe.MatchString(requiresPython) {
		return "", fmt.Errorf("requires-python %q has no lower bound to install from", requiresPython)
	}
	return fmt.Sprintf("%s.%s", match[1], match[2]), nil
}
