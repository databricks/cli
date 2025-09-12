package deployplan

import "strings"

// ParseResourceKey parses a canonical resource key "resources.<group>.<key>" and returns (group, key).
// Returns ("", "") if the input does not match the expected format.
func ParseResourceKey(full string) (string, string) {
	if !strings.HasPrefix(full, "resources.") {
		return "", ""
	}
	rest := strings.TrimPrefix(full, "resources.")
	parts := strings.SplitN(rest, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}
