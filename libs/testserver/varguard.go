package testserver

import (
	"regexp"
	"strings"
)

// unexpandedVarRegex matches shell-style variable references whose name is all
// uppercase: braced `${FOO}` or bare `$FOO`. Acceptance scripts inject env/test
// variables by this convention (UNIQUE_NAME, CURRENT_USER_NAME, JOB_ID, ...).
// Restricting to uppercase names deliberately ignores DABs interpolation, which
// is always lowercase and namespaced (${resources.jobs.foo.id}, ${var.catalog},
// ${bundle.target}); those legitimately reach the API unresolved in some tests.
// The leading [A-Z_] also excludes non-variable dollar usage like $1, $5, $$.
var unexpandedVarRegex = regexp.MustCompile(`\$\{[A-Z_][A-Z0-9_]*\}|\$[A-Z_][A-Z0-9_]*`)

// allowedDollarTokens are the exact `${FOO}`/`$FOO` tokens known to be
// legitimate in API request payloads. The guard rejects everything else, so
// this list is the deliberate, reviewed escape hatch for intentional cases. It
// is empty today; add entries here (with justification) only when a payload
// genuinely needs an uppercase variable-shaped string.
var allowedDollarTokens = map[string]struct{}{}

// exemptPathPrefixes are request paths whose bodies must not be inspected:
//   - file-content uploads carry opaque bytes (notebooks, source files, secret
//     values) that may contain variable-shaped text;
//   - telemetry serializes the bundle config verbatim, including unresolved
//     interpolation and arbitrary user strings, so it is logging, not a payload.
var exemptPathPrefixes = []string{
	"/api/2.0/workspace/import",
	"/api/2.0/workspace-files/import-file/",
	"/api/2.0/fs/files/",
	"/api/2.0/dbfs/",
	"/api/2.0/secrets/put",
	"/telemetry-ext",
}

// detectUnexpandedVar reports the first uppercase shell/test variable reference
// that reached the test server unexpanded. Acceptance scripts interpolate
// variables like ${UNIQUE_NAME} into API payloads; if one is left inside single
// quotes it is sent literally, which only fails intermittently on cloud (see PR
// #5376). Catching it locally turns that class of bug into a deterministic
// failure.
func detectUnexpandedVar(path, rawQuery string, body []byte) (string, bool) {
	for _, prefix := range exemptPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return "", false
		}
	}

	for _, s := range []string{path, rawQuery, string(body)} {
		for _, match := range unexpandedVarRegex.FindAllString(s, -1) {
			if _, ok := allowedDollarTokens[match]; !ok {
				return match, true
			}
		}
	}
	return "", false
}
