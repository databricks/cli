package testdiff

import (
	"regexp"
	"strings"
)

// RemoveLines drops every line matching pattern from input and returns the remaining text
// along with the dropped lines. Use it for output lines that depend on timing or fault
// injection, so that a test does not have to pin them.
func RemoveLines(input string, pattern *regexp.Regexp) (string, []string) {
	var kept, removed []string
	for line := range strings.SplitSeq(input, "\n") {
		if pattern.MatchString(line) {
			removed = append(removed, line)
		} else {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n"), removed
}
