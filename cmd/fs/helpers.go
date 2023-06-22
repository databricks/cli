package fs

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
)

type Scheme string

const (
	DbfsScheme  = Scheme("dbfs")
	LocalScheme = Scheme("file")
	NoScheme    = Scheme("")
)

func filerForPath(ctx context.Context, fullPath string) (filer.Filer, string, error) {
	parts := strings.SplitN(fullPath, ":/", 2)
	if len(parts) < 2 {
		return nil, "", fmt.Errorf(`no scheme specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar or file:/c:/foo/bar`, fullPath)
	}
	scheme := Scheme(parts[0])
	path := parts[1]
	switch scheme {
	case DbfsScheme:
		w := root.WorkspaceClient(ctx)
		// If the specified path has the "Volumes" prefix, use the Files API.
		if strings.HasPrefix(path, "Volumes/") {
			f, err := filer.NewFilesClient(w, "/")
			return f, path, err
		}
		f, err := filer.NewDbfsClient(w, "/")
		return f, path, err

	case LocalScheme:
		if runtime.GOOS == "windows" {
			parts := strings.SplitN(path, ":", 2)
			if len(parts) < 2 {
				return nil, "", fmt.Errorf("no volume specfied for path: %s", path)
			}
			volume := parts[0] + ":"
			relPath := parts[1]
			f, err := filer.NewLocalClient(volume)
			return f, relPath, err
		}
		f, err := filer.NewLocalClient("/")
		return f, path, err

	default:
		return nil, "", fmt.Errorf(`unsupported scheme %s specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar or file:/c:/foo/bar`, scheme, fullPath)
	}
}

func isDbfsPath(path string) bool {
	return strings.HasPrefix(path, "dbfs:/")
}
