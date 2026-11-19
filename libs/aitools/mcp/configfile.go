package mcp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/databrickscfg"
)

// ResolveConfigFile returns the absolute path of the config file that name refers to. An empty
// name is the default file, and a leading "~" is the home directory. Symlinks are resolved while
// the path is still as the user wrote it: the OS reads "link/../x" through the link's target, so
// the same file has one path however it is named. A path that does not exist is kept as written,
// since there is no file to resolve and cleaning it could name another file.
func ResolveConfigFile(ctx context.Context, name string) (string, error) {
	path, err := databrickscfg.ResolveConfigFilePath(ctx, name)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		return filepath.Abs(resolved)
	case errors.Is(err, fs.ErrNotExist):
		return AbsoluteAsWritten(path)
	default:
		return "", err
	}
}

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
	if strings.HasPrefix(path, string(filepath.Separator)) {
		// A rooted path with no drive, such as \tools on Windows, is on the working directory's drive.
		return filepath.VolumeName(wd) + path, nil
	}
	if strings.HasSuffix(wd, string(filepath.Separator)) {
		// The working directory is a root, which already ends with a separator.
		return wd + path, nil
	}
	return wd + string(filepath.Separator) + path, nil
}

// samePath reports whether two resolved config file paths name the same file. Existing files are
// compared by identity, which also matches hard links and case variants on case-insensitive file
// systems. Paths that do not exist are compared as text.
func samePath(a, b string) bool {
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	if errA == nil && errB == nil {
		return os.SameFile(fa, fb)
	}
	return a == b
}
