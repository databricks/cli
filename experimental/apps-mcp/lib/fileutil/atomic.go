// Package fileutil provides file operation utilities.
package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes content to a file atomically by writing to a temporary file
// first and then renaming it to the target path. This ensures that the file is never
// left in a partially written state.
//
// The function creates parent directories if they don't exist and cleans up the
// temporary file on error.
func AtomicWriteFile(path string, content []byte, perm os.FileMode) error {
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, content, perm); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
