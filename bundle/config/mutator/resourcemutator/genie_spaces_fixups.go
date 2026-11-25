package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/stages"
	"github.com/databricks/cli/libs/diag"
)

func init() {
	// Reads (typed): b.Config.Resources.GenieSpaces (checks genie space configurations)
	// Updates (typed): b.Config.Resources.GenieSpaces[].ParentPath (ensures /Workspace prefix is present)
	// Ensures genie space parent paths have the required /Workspace prefix
	stages.Register(stages.StageAfterDefaults, "genie_spaces", GenieSpaceFixups)
}

type genieSpaceFixups struct{}

func GenieSpaceFixups() bundle.Mutator {
	return &genieSpaceFixups{}
}

func (m *genieSpaceFixups) Name() string {
	return "GenieSpaceFixups"
}

func (m *genieSpaceFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, genieSpace := range b.Config.Resources.GenieSpaces {
		if genieSpace == nil {
			continue
		}

		genieSpace.ParentPath = ensureWorkspacePrefix(genieSpace.ParentPath)
	}

	return nil
}
