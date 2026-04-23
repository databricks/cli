package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
)

type defineDefaultWorkspaceRoot struct{}

// DefineDefaultWorkspaceRoot defaults Workspace.RootPath to
// "~/databricks/ucm/<name>/<target>" when not user-set. Parallel to bundle's
// DefineDefaultWorkspaceRoot. The leading "~" is expanded to
// "/Workspace/Users/<user>" by ExpandWorkspaceRoot in the next step.
func DefineDefaultWorkspaceRoot() ucm.Mutator { return &defineDefaultWorkspaceRoot{} }

func (m *defineDefaultWorkspaceRoot) Name() string { return "DefineDefaultWorkspaceRoot" }

func (m *defineDefaultWorkspaceRoot) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	if u.Config.Workspace.RootPath != "" {
		return nil
	}
	if u.Config.Ucm.Name == "" {
		return diag.Errorf("unable to define default workspace root: ucm.name not defined")
	}
	if u.Config.Ucm.Target == "" {
		return diag.Errorf("unable to define default workspace root: target not selected")
	}

	u.Config.Workspace.RootPath = fmt.Sprintf("~/databricks/ucm/%s/%s", u.Config.Ucm.Name, u.Config.Ucm.Target)
	return nil
}
