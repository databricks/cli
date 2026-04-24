package mutator

import (
	"context"
	"path"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
)

type expandWorkspaceRoot struct{}

// ExpandWorkspaceRoot replaces a leading "~" in Workspace.RootPath with
// "/Workspace/Users/<current-user>". Runs after DefineDefaultWorkspaceRoot and
// PopulateCurrentUser. Mirrors bundle/config/mutator.ExpandWorkspaceRoot.
func ExpandWorkspaceRoot() ucm.Mutator { return &expandWorkspaceRoot{} }

func (m *expandWorkspaceRoot) Name() string { return "ExpandWorkspaceRoot" }

func (m *expandWorkspaceRoot) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	root := u.Config.Workspace.RootPath
	if root == "" {
		return diag.Errorf("unable to expand workspace root: workspace root not defined")
	}

	cur := u.CurrentUser
	if cur == nil || cur.User == nil || cur.UserName == "" {
		return diag.Errorf("unable to expand workspace root: current user not set")
	}

	if strings.HasPrefix(root, "~/") {
		home := "/Workspace/Users/" + cur.UserName
		u.Config.Workspace.RootPath = path.Join(home, root[2:])
	}
	return nil
}
