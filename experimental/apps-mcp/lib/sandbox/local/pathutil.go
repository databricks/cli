package local

import (
	"github.com/databricks/cli/experimental/apps-mcp/lib/pathutil"
)

// ValidatePath ensures that the user-provided path is within baseDir and
// returns the absolute path. This prevents directory traversal attacks.
//
// Deprecated: Use pathutil.ValidatePath instead.
func ValidatePath(baseDir, userPath string) (string, error) {
	return pathutil.ValidatePath(baseDir, userPath)
}

// MustValidatePath is like ValidatePath but panics on error.
// Use this only in tests or when you know the path is safe.
//
// Deprecated: Use pathutil.MustValidatePath instead.
func MustValidatePath(baseDir, userPath string) string {
	return pathutil.MustValidatePath(baseDir, userPath)
}

// RelativePath returns the relative path from baseDir to targetPath.
// Both paths should be absolute. Returns an error if targetPath is not
// within baseDir.
//
// Deprecated: Use pathutil.RelativePath instead.
func RelativePath(baseDir, targetPath string) (string, error) {
	return pathutil.RelativePath(baseDir, targetPath)
}
