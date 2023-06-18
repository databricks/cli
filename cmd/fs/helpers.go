package fs

import (
	"context"
	"fmt"
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
		f, err := filer.NewDbfsClient(w, "/")
		return f, path, err

	case LocalScheme:
		f, err := filer.NewLocalClient("/")
		return f, path, err

	default:
		return nil, "", fmt.Errorf(`unsupported scheme %s specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar or file:/c:/foo/bar`, scheme, fullPath)
	}
}
