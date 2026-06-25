package libraries

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
)

func filerForVolume(ctx context.Context, b *bundle.Bundle, uploadPath string) (filer.Filer, string, error) {
	w := b.WorkspaceClient(ctx)
	f, err := filer.NewFilesClient(w, uploadPath)
	return f, uploadPath, err
}
