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

// underscoreRuns matches two or more "_" in a row.
var underscoreRuns = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`_{2,}`)
})

// Sanitize lowercases s, replaces each run of characters outside [a-z0-9_-] with "-",
// collapses runs of "_" into one, and trims leading and trailing "-" and "_". Agents name
// tools mcp__<server>__<tool> and split on "__", so a part of a server name must not contain
// "__" or start or end with "_".
func Sanitize(s string) string {
	name := invalidNameChars().ReplaceAllString(strings.ToLower(s), "-")
	name = underscoreRuns().ReplaceAllString(name, "_")
	return strings.Trim(name, "-_")
}

// ServerName returns the agent-facing name of a server for a service in location and a profile.
// Tool names in agents are mcp__<name>__<tool>, so the name must stay stable for the entry's life.
func ServerName(location, leaf, profile string) string {
	if strings.EqualFold(location, systemAILocation) {
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
