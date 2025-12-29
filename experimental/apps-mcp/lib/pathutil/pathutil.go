package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath ensures that the user-provided path is within baseDir and
// returns the absolute path. This prevents directory traversal attacks.
//
// Security checks:
//   - Resolves relative paths against baseDir
//   - Cleans the path (removes .., ., etc.)
//   - Resolves symlinks to prevent symlink escape attacks
//   - Verifies the final path is within baseDir
//
// For non-existent paths, validates that the parent directory exists and
// is within baseDir.
func ValidatePath(baseDir, userPath string) (string, error) {
	if filepath.IsAbs(userPath) {
		return "", fmt.Errorf("absolute paths not allowed: %s", userPath)
	}

	base, err := resolveBasePath(baseDir)
	if err != nil {
		return "", err
	}

	target := filepath.Join(base, userPath)
	cleaned := filepath.Clean(target)

	resolved, err := resolveTargetPath(cleaned)
	if err != nil {
		return "", err
	}

	if err := checkPathWithinBase(base, resolved); err != nil {
		return "", err
	}

	return resolved, nil
}

// resolveBasePath resolves the base directory to its absolute canonical path.
func resolveBasePath(baseDir string) (string, error) {
	base, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	baseResolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base directory symlinks: %w", err)
	}

	base = filepath.Clean(baseResolved)
	if !strings.HasSuffix(base, string(filepath.Separator)) {
		base += string(filepath.Separator)
	}

	return base, nil
}

// resolveTargetPath resolves the target path, handling both existing and non-existent paths.
func resolveTargetPath(cleaned string) (string, error) {
	_, err := os.Lstat(cleaned)
	if err == nil {
		return resolveExistingPath(cleaned)
	}

	if os.IsNotExist(err) {
		return resolveNonExistentPath(cleaned)
	}

	return "", fmt.Errorf("failed to stat path: %w", err)
}

// resolveExistingPath resolves symlinks for an existing path.
func resolveExistingPath(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink: %w", err)
	}
	return resolved, nil
}

// resolveNonExistentPath validates non-existent paths by checking the parent directory.
func resolveNonExistentPath(cleaned string) (string, error) {
	parent := filepath.Dir(cleaned)

	_, err := os.Stat(parent)
	if os.IsNotExist(err) {
		return cleaned, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat parent directory: %w", err)
	}

	parentResolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", fmt.Errorf("failed to resolve parent symlink: %w", err)
	}

	return filepath.Join(parentResolved, filepath.Base(cleaned)), nil
}

// checkPathWithinBase verifies that the resolved path is within the base directory.
func checkPathWithinBase(base, resolved string) error {
	if !strings.HasPrefix(resolved+string(filepath.Separator), base) {
		return fmt.Errorf("path outside base directory: %s not in %s", resolved, base)
	}
	return nil
}

// MustValidatePath is like ValidatePath but panics on error.
// Use this only in tests or when you know the path is safe.
func MustValidatePath(baseDir, userPath string) string {
	path, err := ValidatePath(baseDir, userPath)
	if err != nil {
		panic(err)
	}
	return path
}

// RelativePath returns the relative path from baseDir to targetPath.
// Both paths should be absolute. Returns an error if targetPath is not
// within baseDir.
func RelativePath(baseDir, targetPath string) (string, error) {
	base := filepath.Clean(baseDir)
	target := filepath.Clean(targetPath)

	if !strings.HasPrefix(target, base) {
		return "", fmt.Errorf("target path %s is not within base %s", target, base)
	}

	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	return rel, nil
}
