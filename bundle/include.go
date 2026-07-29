package bundle

import (
	"fmt"
	"path/filepath"
	"slices"
)

// SetIncludePatterns records the raw (unexpanded) 'include' patterns from the
// root databricks.yml. ProcessRootIncludes calls this before it overwrites
// Config.Include with the expanded list of loaded files.
func (b *Bundle) SetIncludePatterns(patterns []string) {
	b.includePatterns = patterns
}

// MatchedIncludeFiles returns the absolute paths currently matched by the raw
// root include patterns. The result changes when matching files are created or
// removed after the bundle was loaded.
func (b *Bundle) MatchedIncludeFiles() ([]string, error) {
	seen := make(map[string]struct{})
	for _, pattern := range b.includePatterns {
		if filepath.IsAbs(pattern) {
			return nil, fmt.Errorf("include pattern %s is absolute", pattern)
		}
		matches, err := filepath.Glob(filepath.Join(b.BundleRootPath, pattern))
		if err != nil {
			return nil, fmt.Errorf("matching include pattern %s: %w", pattern, err)
		}
		for _, match := range matches {
			absolute, err := filepath.Abs(match)
			if err != nil {
				return nil, fmt.Errorf("resolving included file %s: %w", match, err)
			}
			seen[absolute] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for path := range seen {
		result = append(result, path)
	}
	slices.Sort(result)
	return result, nil
}

// IsFileIncluded reports whether absPath is matched by any of the root
// 'include' patterns. absPath must be an absolute path. Patterns are resolved
// against the files currently on disk, so a file generated after load is
// matched only if it now satisfies one of the patterns.
func (b *Bundle) IsFileIncluded(absPath string) bool {
	for _, pattern := range b.includePatterns {
		// Includes must be relative paths; the loader errors on absolute ones.
		if filepath.IsAbs(pattern) {
			continue
		}

		// Anchor the pattern to the bundle root, same as ProcessRootIncludes.
		matches, err := filepath.Glob(filepath.Join(b.BundleRootPath, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			absMatch, err := filepath.Abs(match)
			if err != nil {
				continue
			}
			if absMatch == absPath {
				return true
			}
		}
	}

	return false
}
