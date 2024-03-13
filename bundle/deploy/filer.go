package deploy

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
)

// FilerFactory is a function that returns a filer.Filer.
type FilerFactory func(b *bundle.Bundle) (filer.Filer, error)

// StateFiler returns a filer.Filer that can be used to read/write state files.
func StateFiler(b *bundle.Bundle) (filer.Filer, error) {
	return filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
}
