package libraries

import (
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

func filerForVolume(b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	w := b.WorkspaceClient()
	uploadPath := path.Join(b.Config.Workspace.ArtifactPath, InternalDirName)
	f, err := filer.NewFilesClient(w, uploadPath)
	return f, uploadPath, diag.FromErr(err)
}
