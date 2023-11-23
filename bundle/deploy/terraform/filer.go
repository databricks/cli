package terraform

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
)

// filerFunc is a function that returns a filer.Filer.
type filerFunc func(b *bundle.Bundle) (filer.Filer, error)

// stateFiler returns a filer.Filer that can be used to read/write state files.
func stateFiler(b *bundle.Bundle) (filer.Filer, error) {
	return filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
}

// identityFiler returns a filerFunc that returns the specified filer.
func identityFiler(f filer.Filer) filerFunc {
	return func(_ *bundle.Bundle) (filer.Filer, error) {
		return f, nil
	}
}
