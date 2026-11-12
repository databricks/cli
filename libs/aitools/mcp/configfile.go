package mcp

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/databrickscfg"
)

// foldCase reports whether config file paths compare case-insensitively. Windows and macOS
// file systems are case-insensitive by default. Tests set it to cover both comparisons.
var foldCase = runtime.GOOS == "windows" || runtime.GOOS == "darwin"

// resolveConfigFile returns the absolute path of the config file that name refers to. An empty
// name is the default file, and a leading "~" is the home directory. Symlinks are resolved while
// the path is still as the user wrote it: the OS reads "link/../x" through the link's target, so
// the same file has one path however it is named. A path that does not exist is made absolute as
// written, since there is no file to resolve.
func resolveConfigFile(ctx context.Context, name string) (string, error) {
	path, err := databrickscfg.ResolveConfigFilePath(ctx, name)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		path = resolved
	case !errors.Is(err, fs.ErrNotExist):
		return "", err
	}
	return filepath.Abs(path)
}

// samePath reports whether two resolved config file paths name the same file.
func samePath(a, b string) bool {
	if foldCase {
		return strings.EqualFold(a, b)
	}
	return a == b
}
