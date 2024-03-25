package mutator

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type expandWorkspaceRoot struct{}

// ExpandWorkspaceRoot expands ~ if present in the workspace root.
func ExpandWorkspaceRoot() bundle.Mutator {
	return &expandWorkspaceRoot{}
}

func (m *expandWorkspaceRoot) Name() string {
	return "ExpandWorkspaceRoot"
}

func (m *expandWorkspaceRoot) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	root := b.Config.Workspace.RootPath
	if root == "" {
		return diag.Errorf("unable to expand workspace root: workspace root not defined")
	}

	currentUser := b.Config.Workspace.CurrentUser
	if currentUser == nil || currentUser.UserName == "" {
		return diag.Errorf("unable to expand workspace root: current user not set")
	}

	if strings.HasPrefix(root, "~/") {
		home := fmt.Sprintf("/Users/%s", currentUser.UserName)
		b.Config.Workspace.RootPath = path.Join(home, root[2:])
	}

	return nil
}
