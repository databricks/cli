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

func setupFiler(ctx context.Context, path string) (filer.Filer, error) {
	w := root.WorkspaceClient(ctx)
	scheme := resolveScheme(path)
	resolvedPath, err := resolvePath(path, scheme)
	if err != nil {
		return nil, err
	}

	switch scheme {
	case DbfsScheme:
		return filer.NewDbfsClient(w, resolvedPath)
	case LocalScheme:
		return filer.NewLocalClient(resolvedPath)
	case NoScheme:
		return nil, fmt.Errorf(`no scheme specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar`, path)

	default:
		return nil, fmt.Errorf("scheme %q is not supported", scheme)
	}

}

func resolvePath(path string, scheme Scheme) (string, error) {
	if scheme == NoScheme {
		return path, nil
	}
	prefix := string(scheme) + ":"
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("expected path with scheme %s. Example: %s:/foo/bar", scheme, scheme)
	}
	return strings.TrimPrefix(path, prefix), nil
}

func resolveScheme(path string) Scheme {
	parts := strings.Split(path, ":")
	if len(parts) < 2 {
		return NoScheme
	}
	return Scheme(parts[0])
}
