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

// ServerName returns the agent-facing name of the server for a service (catalog.schema.leaf)
// and a profile: <leaf>-<profile> in system.ai, <catalog>-<schema>-<leaf>-<profile> elsewhere.
// Each part is sanitized and parts that sanitize to nothing are left out. Tool names in
// agents are mcp__<name>__<tool>, so the name must stay stable for the entry's life.
func ServerName(fullName, profile string) string {
	parts := strings.Split(fullName, ".")
	if inSystemAI(fullName) {
		parts = []string{LeafOf(fullName)}
	}

	var name []string
	for _, part := range append(parts, profile) {
		if s := Sanitize(part); s != "" {
			name = append(name, s)
		}
	}
	return strings.Join(name, "-")
}

// leafOf returns the service name after the last ".". Unity Catalog names cannot contain ".".
func LeafOf(fullName string) string {
	return fullName[strings.LastIndex(fullName, ".")+1:]
}

// inSystemAI reports whether a service full name is in system.ai, in any casing.
func inSystemAI(fullName string) bool {
	return strings.HasPrefix(strings.ToLower(fullName), SystemAILocation+".")
}

// UniqueName returns base, or base-2, base-3, ... for the first candidate for which taken
// returns false.
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
