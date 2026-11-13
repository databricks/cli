package mcpconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	return joinWorkingDir(path, wd), nil
}

// joinWorkingDir joins a relative path to wd. A root-relative path, one that starts with a separator but has no volume
// (such as "\claude" on Windows), is on the volume of wd, so it is joined to that volume, not to the whole directory.
func joinWorkingDir(path, wd string) string {
	if strings.HasPrefix(path, string(filepath.Separator)) {
		return filepath.VolumeName(wd) + path
	}
	return wd + string(filepath.Separator) + path
}
