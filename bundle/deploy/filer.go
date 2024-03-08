package deploy

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
)

// filerFactory is a function that returns a filer.Filer.
type filerFactory func(b *bundle.Bundle) (filer.Filer, error)

// stateFiler returns a filer.Filer that can be used to read/write state files.
func stateFiler(b *bundle.Bundle) (filer.Filer, error) {
	return filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
}
