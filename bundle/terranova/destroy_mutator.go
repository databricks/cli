package terranova

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dag"
	"github.com/databricks/cli/libs/diag"
)

type terranovaDestroyMutator struct{}

func TerranovaDestroy() bundle.Mutator {
	return &terranovaDestroyMutator{}
}

func (m *terranovaDestroyMutator) Name() string {
	return "TerranovaDestroy"
}

func (m *terranovaDestroyMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.SafeDiagnostics
	client := b.WorkspaceClient()

	allResources := b.ResourceDatabase.GetAllResources()
	g := dag.NewGraph[tnstate.ResourceNode]()

	for _, node := range allResources {
		g.AddNode(node)
	}

	// TODO: respect dependencies; dependencies need to be part of state, not config.

	err := g.Run(maxPoolSize, func(node tnstate.ResourceNode) {
		err := tnresources.DestroyResource(ctx, client, node.Section, node.ID)
		if err != nil {
			diags.AppendErrorf("destroying %s: %s", node, err)
			return
		}
		// TODO: did DestroyResource fail because it did not exist? we can clean it up from the state as well

		err = b.ResourceDatabase.DeleteState(node.Section, node.Name)
		if err != nil {
			diags.AppendErrorf("deleting from the state %s: %s", node, err)
			return
		}
	})
	if err != nil {
		diags.AppendError(err)
	}

	_ = b.ResourceDatabase.Finalize()

	return diags.Diags
}
