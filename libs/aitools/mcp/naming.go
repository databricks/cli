package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// MaxServerNameLen caps a server name. Agents expose tools as mcp__<server>__<tool> and limit
// tool names to about 64 characters.
const MaxServerNameLen = 32

// serverNameHashLen is the number of hex characters that end a capped server name.
const serverNameHashLen = 6

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
// Each part is sanitized and parts that sanitize to nothing are left out, except the leaf, which
// makes an error wrapping ErrNoServerName. A name longer than MaxServerNameLen is capped by
// capServerName. Tool names in agents are mcp__<name>__<tool>, so the name must stay stable for
// the entry's life.
func ServerName(fullName, profile string) (string, error) {
	leaf := LeafOf(fullName)
	if Sanitize(leaf) == "" {
		return "", fmt.Errorf("%w: %q has no characters that can be used in a server name", ErrNoServerName, leaf)
	}

	parts := strings.Split(fullName, ".")
	if inSystemAI(fullName) {
		parts = []string{leaf}
	}

	var name []string
	for _, part := range append(parts, profile) {
		if s := Sanitize(part); s != "" {
			name = append(name, s)
		}
	}
	return capServerName(strings.Join(name, "-")), nil
}

// capServerName returns name unchanged when it fits in MaxServerNameLen. Otherwise it cuts name
// to fit beside "-" and serverNameHashLen hex characters of the sha256 of the uncut name, drops
// any "-" or "_" the cut leaves at its end, and appends "-" and the hash. Long names that share a
// prefix stay distinct, and the same name always gets the same result. name is sanitized, so it
// is ASCII and cutting it by byte is safe.
func capServerName(name string) string {
	if len(name) <= MaxServerNameLen {
		return name
	}
	sum := sha256.Sum256([]byte(name))
	hash := hex.EncodeToString(sum[:])[:serverNameHashLen]
	head := strings.TrimRight(name[:MaxServerNameLen-len(hash)-1], "-_")
	return head + "-" + hash
}

// ErrNoServerName is returned by ServerName when the leaf of a service name has no characters
// that a server name can contain. The user has to choose the server name instead.
var ErrNoServerName = errors.New("cannot derive a server name from the service name")

// LeafOf returns the service name after the last ".". Unity Catalog names cannot contain ".".
func LeafOf(fullName string) string {
	return fullName[strings.LastIndex(fullName, ".")+1:]
}

// inSystemAI reports whether a service full name is in system.ai, in any casing.
func inSystemAI(fullName string) bool {
	return strings.HasPrefix(strings.ToLower(fullName), SystemAILocation+".")
}

// UniqueName returns base, or base-2, base-3, ... for the first candidate for which taken
// returns false. Each candidate is capped like a server name, so a suffix cannot push the name
// past MaxServerNameLen.
func UniqueName(base string, taken func(string) bool) string {
	if !taken(base) {
		return base
	}
	for n := 2; ; n++ {
		candidate := capServerName(fmt.Sprintf("%s-%d", base, n))
		if !taken(candidate) {
			return candidate
		}
	}
}
