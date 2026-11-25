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

func (m *genieSpaceFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, genieSpace := range b.Config.Resources.GenieSpaces {
		if genieSpace == nil {
			continue
		}

		genieSpace.ParentPath = ensureWorkspacePrefix(genieSpace.ParentPath)
	}

	return nil
}
