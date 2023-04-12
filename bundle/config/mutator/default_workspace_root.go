package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
)

type defineDefaultWorkspaceRoot struct{}

// DefineDefaultWorkspaceRoot defines the default workspace root path.
func DefineDefaultWorkspaceRoot() bundle.Mutator {
	return &defineDefaultWorkspaceRoot{}
}

func (m *defineDefaultWorkspaceRoot) Name() string {
	return "DefineDefaultWorkspaceRoot"
}

func (m *defineDefaultWorkspaceRoot) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	if b.Config.Workspace.RootPath != "" {
		return nil, nil
	}

	if b.Config.Bundle.Name == "" {
		return nil, fmt.Errorf("unable to define default workspace root: bundle name not defined")
	}

	if b.Config.Bundle.Environment == "" {
		return nil, fmt.Errorf("unable to define default workspace root: bundle environment not selected")
	}

	b.Config.Workspace.RootPath = fmt.Sprintf(
		"~/.bundle/%s/%s",
		b.Config.Bundle.Name,
		b.Config.Bundle.Environment,
	)
	return nil, nil
}
