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

func resolveDbfsPath(path string) (string, error) {
	if !strings.HasPrefix(path, "dbfs:/") {
		return "", fmt.Errorf("expected dbfs path (with the dbfs:/ prefix): %s", path)
	}

	return strings.TrimPrefix(path, "dbfs:"), nil
}

func filerForScheme(ctx context.Context, scheme Scheme) (filer.Filer, error) {
	switch scheme {
	case DbfsScheme:
		w := root.WorkspaceClient(ctx)
		return filer.NewDbfsClient(w, "/")
	case LocalScheme:
		return filer.NewLocalClient("/")
	default:
		return nil, fmt.Errorf("scheme %q is not supported", scheme)
	}
}

func removeScheme(path string, scheme Scheme) (string, error) {
	if scheme == NoScheme {
		return path, nil
	}
	prefix := string(scheme) + ":"
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("expected path %s to have scheme %s. Example: %s:/foo/bar", path, scheme, scheme)
	}
	return strings.TrimPrefix(path, prefix), nil
}

func scheme(path string) Scheme {
	parts := strings.Split(path, ":")
	if len(parts) < 2 {
		return NoScheme
	}
	return Scheme(parts[0])
}
