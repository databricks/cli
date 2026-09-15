package patchwheel

import (
	"regexp"
	"strings"
)

// Matches a pip extras suffix right after ".whl", e.g. "[train]" in "pkg.whl[train]".
// Anchored so glob classes elsewhere (e.g. "pkg[12].whl") are left alone.
var wheelExtrasRegex = regexp.MustCompile(`(?i)\.whl(\[[^\]]*\])$`)

// SplitWheelExtras splits a wheel path into its base and pip extras suffix (e.g. "[train]").
func SplitWheelExtras(path string) (string, string) {
	m := wheelExtrasRegex.FindStringSubmatch(path)
	if m == nil {
		return path, ""
	}
	extras := m[1]
	return strings.TrimSuffix(path, extras), extras
}
