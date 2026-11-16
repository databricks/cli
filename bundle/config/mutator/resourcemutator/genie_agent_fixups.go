package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type genieAgentFixups struct{}

func GenieAgentFixups() bundle.Mutator {
	return &genieAgentFixups{}
}

func (m *genieAgentFixups) Name() string {
	return "GenieAgentFixups"
}

func (m *genieAgentFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, genieAgent := range b.Config.Resources.GenieAgents {
		if genieAgent == nil {
			continue
		}

		genieAgent.ParentPath = ensureWorkspacePrefix(genieAgent.ParentPath)
	}

	return nil
}
