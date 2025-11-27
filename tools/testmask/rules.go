package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

// AllRules returns all rules in order of application.
func AllRules() []RuleFunc {
	return []RuleFunc{
		handleExperimental,
	}
}

// longestCommonDirectoryPrefix returns the longest common directory prefix of a list of files.
// It always returns a directory path, never a file path.
func longestCommonDirectoryPrefix(files []string) string {
	if len(files) == 0 {
		return ""
	}
	if len(files) == 1 {
		dir := filepath.Dir(files[0])
		if dir == "." {
			return ""
		}
		return dir + "/"
	}
	prefix := files[0]
	for _, file := range files[1:] {
		prefix = longestCommonPrefix(prefix, file)
		if prefix == "" {
			return ""
		}
	}
	// Ensure we return a directory path (remove filename if present)
	if lastSlash := strings.LastIndex(prefix, "/"); lastSlash >= 0 {
		return prefix[:lastSlash+1]
	}
	return ""
}

// longestCommonPrefix returns the longest common prefix of two strings.
// For file paths, it stops at directory boundaries (after a "/").
func longestCommonPrefix(a, b string) string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	lastSlash := -1
	for i := range minLen {
		if a[i] != b[i] {
			if lastSlash >= 0 {
				return a[:lastSlash+1]
			}
			return ""
		}
		if a[i] == '/' {
			lastSlash = i
		}
	}
	if lastSlash >= 0 {
		return a[:lastSlash+1]
	}
	return a[:minLen]
}

// handleExperimental deals with changes under the experimental/ directory.
func handleExperimental(files []string) ([]string, []string, error) {
	prefix := longestCommonDirectoryPrefix(files)

	// Match this prefix to experimental and optionally a package nested under it.
	pkg := regexp.MustCompile(`^experimental(/[\w-_]+)?`).FindString(prefix)
	if pkg == "" {
		return nil, nil, errSkip
	}

	return []string{pkg + "/..."}, []string{}, nil
}
