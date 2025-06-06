package terranova

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
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
	g := dagrun.NewGraph[tnstate.ResourceNode]()

	for _, node := range allResources {
		g.AddNode(node)
	}

	// TODO: do we need to respect dependencies here? Should we store them in state in that case?
	// Alternatively, do a few rounds of delete to make we capture cases where A needs to be deleted before B.

	err := g.Run(defaultParallelism, func(node tnstate.ResourceNode) {
		err := tnresources.DestroyResource(ctx, client, node.Section, node.ID)
		if err != nil {
			diags.AppendErrorf("destroying %s: %s", node, err)
			return
		}
		// TODO: handle situation where resources is already gone for whatever reason.

		err = b.ResourceDatabase.DeleteState(node.Section, node.Name)
		if err != nil {
			diags.AppendErrorf("deleting from the state %s: %s", node, err)
			return
		}
	})
	if err != nil {
		diags.AppendError(err)
	}

	err = b.ResourceDatabase.Finalize()
	if err != nil {
		diags.AppendError(err)
	}

	return diags.Diags
}
