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

func filerForPath(ctx context.Context, path string) (filer.Filer, string, error) {
	parts := strings.Split(path, ":")
	if len(parts) < 2 {
		return nil, "", fmt.Errorf(`no scheme specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar`, path)
	}
	scheme := Scheme(parts[0])
	switch scheme {
	case DbfsScheme:
		w := root.WorkspaceClient(ctx)
		cleanPath, err := trimDbfsScheme(path)
		if err != nil {
			return nil, "", err
		}
		f, err := filer.NewDbfsClient(w, "/")
		return f, cleanPath, err

	case LocalScheme:
		cleanPath, err := trimLocalScheme(path)
		if err != nil {
			return nil, "", err
		}
		f, err := filer.NewLocalClient("/")
		return f, cleanPath, err

	default:
		return nil, "", fmt.Errorf(`unsupported scheme %s specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar`, scheme, path)
	}
}

func trimDbfsScheme(path string) (string, error) {
	if !strings.HasPrefix(path, "dbfs:/") {
		return "", fmt.Errorf("expected dbfs path (with the dbfs:/ prefix): %s", path)
	}

	return strings.TrimPrefix(path, "dbfs:"), nil
}

func trimLocalScheme(path string) (string, error) {
	if !strings.HasPrefix(path, "file:") {
		return "", fmt.Errorf("expected file path (with the file: prefix): %s", path)
	}

	return strings.TrimPrefix(path, "file:"), nil
}
