package localenv

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// clauseRe splits a single requires-python clause into its operator (optional),
// MAJOR.MINOR version, and an optional patch component. A clause with no operator
// is a bare floor. The patch capture (group 4) is needed to interpret a strict
// ">" correctly: ">3.10" excludes all of 3.10.x, but ">3.10.5" is still satisfied
// by 3.10.6, so only the former bumps the minor.
var clauseRe = regexp.MustCompile(`^(>=|<=|===|==|~=|!=|<|>)?\s*(\d+)\.(\d+)(\.\d+)?`)

// defaultServerlessVersion is the serverless environment version used when the
// source (a serverless job with no recorded version, or a bundle that only
// records "serverless") does not pin one. It is a documented stand-in for the
// latest LTS (spec §4.3 / §target-resolution); VS Code resolves the real version
// itself and passes --serverless-version explicitly, so this fallback only
// applies when the version is genuinely unknown.
const defaultServerlessVersion = "5"

// serverlessVersionRe matches an accepted --serverless-version input: a bare
// number ("5", the documented form) or a "v"-prefixed one ("v5"/"V5").
var serverlessVersionRe = regexp.MustCompile(`^[vV]?[0-9]+$`)

// ValidServerlessVersion reports whether s is an accepted serverless version
// input. It is validated at resolve time so a malformed value (e.g. "vv5", "v",
// " 5") fails fast with an actionable error rather than resolving to a bogus
// env key that only 404s two phases later at fetch.
func ValidServerlessVersion(s string) bool {
	return serverlessVersionRe.MatchString(s)
}

// NormalizeServerless returns the canonical "vN" spelling of a serverless
// version. The documented input is a bare number ("5"), but a "v"-prefixed form
// ("v5"/"V5") is also accepted; both map to the "serverless-vN" env key.
// Callers should validate with ValidServerlessVersion first.
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

// PythonMinorFromRequires parses a PEP 440 requires-python string and returns
// the MAJOR.MINOR of the Python version to install: the effective lower bound.
//
// A requires-python is a comma-separated list of clauses in any order (e.g.
// "<3.13,>=3.10"). Each clause is classified by operator:
//   - lower-bound / pinning (>=, >, ==, ~=, ===) or a bare MAJOR.MINOR with no
//     operator establishes a floor;
//   - upper-bound / exclusion (<, <=, !=) does not — those versions are capped
//     or forbidden and must never be installed.
//
// The result is the highest floor across all floor clauses (so ">=3.8,>=3.11"
// yields 3.11, the version that satisfies every clause). A spec with no floor
// clause at all (e.g. "<3.13" or "!=3.12") is an error rather than a guess.
func PythonMinorFromRequires(requiresPython string) (string, error) {
	bestMajor, bestMinor := -1, -1
	sawClause := false
	for clause := range strings.SplitSeq(requiresPython, ",") {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		m := clauseRe.FindStringSubmatch(clause)
		if m == nil {
			continue
		}
		sawClause = true
		op := m[1]
		// Upper-bound and exclusion operators never establish a floor.
		if op == "<" || op == "<=" || op == "!=" {
			continue
		}
		major, _ := strconv.Atoi(m[2])
		minor, _ := strconv.Atoi(m[3])
		hasPatch := m[4] != ""
		// A strict ">" with no patch excludes the whole given minor series (PEP 440:
		// ">3.10" matches neither 3.10 nor any 3.10.x), so the lowest installable
		// minor is the next one up. But ">3.10.5" is still satisfied by 3.10.6, so a
		// patch-qualified strict bound leaves the minor unchanged.
		if op == ">" && !hasPatch {
			minor++
		}
		if major > bestMajor || (major == bestMajor && minor > bestMinor) {
			bestMajor, bestMinor = major, minor
		}
	}
	if bestMajor >= 0 {
		return fmt.Sprintf("%d.%d", bestMajor, bestMinor), nil
	}
	if sawClause {
		return "", fmt.Errorf("requires-python %q has no lower bound to install from", requiresPython)
	}
	return "", fmt.Errorf("cannot parse python version from %q", requiresPython)
}
