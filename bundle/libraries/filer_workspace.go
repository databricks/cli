package libraries

import (
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

// We upload artifacts to the workspace in a directory named ".internal" to have
// a well defined location for artifacts that have been uploaded by the DABs.
const InternalDirName = ".internal"

func filerForWorkspace(b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	uploadPath := path.Join(b.Config.Workspace.ArtifactPath, InternalDirName)
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), uploadPath)
	return f, uploadPath, diag.FromErr(err)
}
