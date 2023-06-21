package fs

import (
	"context"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
)

func filerForPath(ctx context.Context, fullPath string) (filer.Filer, string, error) {
	// Split path at : to detect any file schemes
	parts := strings.SplitN(fullPath, ":", 2)

	// If dbfs file scheme is not specified, then it's a local path
	if len(parts) < 2 || parts[0] != "dbfs" {
		f, err := filer.NewLocalClient("")
		return f, fullPath, err
	}

	path := parts[1]
	w := root.WorkspaceClient(ctx)

	// If the specified path has the "Volumes" prefix, use the Files API.
	if strings.HasPrefix(path, "/Volumes/") {
		f, err := filer.NewFilesClient(w, "/")
		return f, path, err
	}

	// The file is a dbfs file, and uses the DBFS APIs
	f, err := filer.NewDbfsClient(w, "/")
	return f, path, err
}
