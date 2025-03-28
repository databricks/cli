package libraries

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

func filerForVolume(b *bundle.Bundle, uploadPath string) (filer.Filer, string, diag.Diagnostics) {
	w := b.WorkspaceClient()
	f, err := filer.NewFilesClient(w, uploadPath)
	return f, uploadPath, diag.FromErr(err)
}
