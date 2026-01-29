package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type genieSpaceFixups struct{}

func GenieSpaceFixups() bundle.Mutator {
	return &genieSpaceFixups{}
}

func (m *genieSpaceFixups) Name() string {
	return "GenieSpaceFixups"
}

// Apply ensures the parent_path has the /Workspace prefix to match what the API returns.
// This prevents persistent recreates when comparing local config vs remote state.
func (m *genieSpaceFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, genieSpace := range b.Config.Resources.GenieSpaces {
		if genieSpace == nil {
			continue
		}

		// Reuse the ensureWorkspacePrefix function from dashboard_fixups.go
		genieSpace.ParentPath = ensureWorkspacePrefix(genieSpace.ParentPath)
	}

	return nil
}
