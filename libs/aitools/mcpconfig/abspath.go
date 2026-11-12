package mcpconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

// AbsoluteAsWritten makes path absolute by joining it to the working directory, without cleaning it, so that a
// ".." after a symlink still resolves through the link's target. Slashes become the OS separator, as cleaning would
// make them: "~/.databrickscfg" expands to the home directory joined with "/" on Windows too.
func AbsoluteAsWritten(path string) (string, error) {
	path = filepath.FromSlash(path)
	if filepath.IsAbs(path) {
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	return wd + string(filepath.Separator) + path, nil
}
