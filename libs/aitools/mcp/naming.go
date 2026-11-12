package mcp

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// invalidNameChars matches each run of characters that a server name cannot contain.
var invalidNameChars = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`[^a-z0-9_-]+`)
})

// Sanitize lowercases s, replaces each run of characters outside [a-z0-9_-] with "-", and
// trims leading and trailing "-".
func Sanitize(s string) string {
	name := invalidNameChars().ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(name, "-")
}

// ServerName returns the agent-facing name of a server for a service in location and a profile.
// Tool names in agents are mcp__<name>__<tool>, so the name must stay stable for the entry's life.
func ServerName(location, leaf, profile string) string {
	if location == systemAILocation {
		return Sanitize(leaf) + "-" + Sanitize(profile)
	}
	return Sanitize(location) + "-" + Sanitize(leaf) + "-" + Sanitize(profile)
}

// UniqueName returns base, or base-2, base-3, ... for the first name that taken reports as free.
func UniqueName(base string, taken func(string) bool) string {
	if !taken(base) {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if !taken(candidate) {
			return candidate
		}
	}
}
