package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type defineDefaultWorkspaceRoot struct{}

// DefineDefaultWorkspaceRoot defines the default workspace root path.
func DefineDefaultWorkspaceRoot() bundle.Mutator {
	return &defineDefaultWorkspaceRoot{}
}

func (m *defineDefaultWorkspaceRoot) Name() string {
	return "DefineDefaultWorkspaceRoot"
}

func (m *defineDefaultWorkspaceRoot) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.RootPath != "" {
		return nil
	}

	// FIXME: this shouldn't appear here
	if b.Config.Project.Name != "" {
		if b.Config.Bundle.Name != "" {
			return diag.Errorf("project and bundle cannot both be set")
		}
		// TODO: properly copy all values from project to bundle
		b.Config.Bundle.Name = b.Config.Project.Name
	}

	if b.Config.Bundle.Name == "" {
		return diag.Errorf("unable to define default workspace root: bundle name not defined")
	}

	if b.Config.Bundle.Target == "" {
		return diag.Errorf("unable to define default workspace root: bundle target not selected")
	}

	prefix := "~/"
	if b.Config.Owner != "" {
		prefix = "/Workspace/Users/" + b.Config.Owner
	}
	b.Config.Workspace.RootPath = fmt.Sprintf(
		prefix+"/.bundle/%s/%s",
		b.Config.Bundle.Name,
		b.Config.Bundle.Target,
	)
	return nil
}
