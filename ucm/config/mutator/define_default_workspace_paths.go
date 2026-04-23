package mutator

import (
	"context"
	"path"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
)

type defineDefaultWorkspacePaths struct{}

// DefineDefaultWorkspacePaths sets workspace paths if they aren't already set.
func DefineDefaultWorkspacePaths() ucm.Mutator {
	return &defineDefaultWorkspacePaths{}
}

func (m *defineDefaultWorkspacePaths) Name() string {
	return "DefaultWorkspacePaths"
}

func (m *defineDefaultWorkspacePaths) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	root := u.Config.Workspace.RootPath
	if root == "" {
		return diag.Errorf("unable to define default workspace paths: workspace root not defined")
	}

	if u.Config.Workspace.StatePath == "" {
		u.Config.Workspace.StatePath = path.Join(root, "state")
	}

	return nil
}
