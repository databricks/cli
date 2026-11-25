package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

func init() {
	// Reads and updates (typed): resources.clusters.*
	registerStageMutator(StageAfterUCDependencies, "clusters", ClusterFixups)
}

type clusterFixups struct{}

func ClusterFixups() bundle.Mutator {
	return &clusterFixups{}
}

func (m *clusterFixups) Name() string {
	return "ClusterFixups"
}

func (m *clusterFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, cluster := range b.Config.Resources.Clusters {
		if cluster == nil {
			continue
		}
		// TODO: we should raise a warning when user specify both InstancePoolId and NodeTypeId since it's illegal.
		// Once we remove TF backend and had warning for some time, we can remove this transformation, the backend
		// will reject such configs.
		ModifyRequestOnInstancePool(&cluster.ClusterSpec)
	}
	return nil
}
