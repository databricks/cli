package libraries

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
)

func filerForWorkspace(ctx context.Context, b *bundle.Bundle, uploadPath string) (filer.Filer, string, error) {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), uploadPath)
	return f, uploadPath, err
}
