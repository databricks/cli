package libraries

import (
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

func filerForWorkspace(b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	uploadPath := path.Join(b.Config.Workspace.ArtifactPath, InternalDirName)
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), uploadPath)
	return f, uploadPath, diag.FromErr(err)
}
