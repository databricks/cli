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
	if volume := filepath.VolumeName(path); volume != "" {
		// A drive-relative path, such as C:cfg on Windows, is relative to the working directory when that is on its drive.
		// Another drive's working directory is not known here, so that path is refused.
		if !strings.EqualFold(volume, filepath.VolumeName(wd)) {
			return "", fmt.Errorf("%s is relative to drive %s, but the working directory is on drive %s", filepath.ToSlash(path), volume, filepath.VolumeName(wd))
		}
		path = path[len(volume):]
	}
	return joinWorkingDir(path, wd), nil
}

// joinWorkingDir joins a relative path to wd. A root-relative path, one that starts with a separator but has no volume
// (such as "\claude" on Windows), is on the volume of wd, so it is joined to that volume, not to the whole directory.
func joinWorkingDir(path, wd string) string {
	if strings.HasPrefix(path, string(filepath.Separator)) {
		return filepath.VolumeName(wd) + path
	}
	if strings.HasSuffix(wd, string(filepath.Separator)) {
		// The working directory is a root, which already ends with a separator.
		return wd + path
	}
	return wd + string(filepath.Separator) + path
}
