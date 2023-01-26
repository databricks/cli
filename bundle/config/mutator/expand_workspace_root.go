package mutator

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/bricks/bundle"
)

type expandWorkspaceRoot struct{}

// ExpandWorkspaceRoot expands ~ if present in the workspace root.
func ExpandWorkspaceRoot() bundle.Mutator {
	return &expandWorkspaceRoot{}
}

func (m *expandWorkspaceRoot) Name() string {
	return "ExpandWorkspaceRoot"
}

func (m *expandWorkspaceRoot) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	root := b.Config.Workspace.Root
	if root == "" {
		return nil, fmt.Errorf("unable to expand workspace root: workspace root not defined")
	}

	currentUser := b.Config.Workspace.CurrentUser
	if currentUser == nil || currentUser.UserName == "" {
		return nil, fmt.Errorf("unable to expand workspace root: current user not set")
	}

	if strings.HasPrefix(root, "~/") {
		home := fmt.Sprintf("/Users/%s", currentUser.UserName)
		b.Config.Workspace.Root = path.Join(home, root[2:])
	}

	return nil, nil
}
