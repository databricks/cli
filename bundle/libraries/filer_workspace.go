package libraries

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

func filerForWorkspace(ctx context.Context, b *bundle.Bundle, uploadPath string) (filer.Filer, string, diag.Diagnostics) {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), uploadPath)
	return f, uploadPath, diag.FromErr(err)
}
