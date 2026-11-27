package mcp

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
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
		return mcpconfig.AbsoluteAsWritten(path)
	default:
		return "", err
	}
}

// SamePath reports whether two resolved config file paths name the same file. Existing files are
// compared by identity, which also matches hard links and case variants on case-insensitive file
// systems. Paths that do not exist are compared as text.
func SamePath(a, b string) bool {
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	if errA == nil && errB == nil {
		return os.SameFile(fa, fb)
	}
	return a == b
}
