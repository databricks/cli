package textutil

import "strings"

// SplitAndTrim splits a comma-separated string, trims whitespace from each
// element, and drops empty entries. It returns nil when no entries remain.
func SplitAndTrim(scopes string) []string {
	var result []string
	for s := range strings.SplitSeq(scopes, ",") {
		scope := strings.TrimSpace(s)
		if scope == "" {
			continue
		}
		result = append(result, scope)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
