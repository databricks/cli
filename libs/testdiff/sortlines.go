package testdiff

import (
	"regexp"
	"slices"
	"strings"
)

// SortLineRuns sorts each run of consecutive lines matching pattern, leaving every
// other line where it is. Use it for output whose line order is genuinely
// nondeterministic — for example "bundle deploy" reporting resources as they are
// applied, which happens in parallel — so that a test pins the set of lines without
// pinning an order the command does not guarantee.
func SortLineRuns(input string, pattern *regexp.Regexp) string {
	lines := strings.Split(input, "\n")
	start := -1
	for i := range lines {
		if pattern.MatchString(lines[i]) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			slices.Sort(lines[start:i])
			start = -1
		}
	}
	if start >= 0 {
		slices.Sort(lines[start:])
	}
	return strings.Join(lines, "\n")
}
