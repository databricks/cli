package validate

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// PipelineClusterSpec rejects cluster settings when serverless compute is enabled.
func PipelineClusterSpec() bundle.ReadOnlyMutator {
	return &pipelineClusterSpec{}
}

type pipelineClusterSpec struct{ bundle.RO }

func (v *pipelineClusterSpec) Name() string {
	return "validate:pipeline_cluster_spec"
}

func (v *pipelineClusterSpec) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, name := range slices.Sorted(maps.Keys(b.Config.Resources.Pipelines)) {
		pipeline := b.Config.Resources.Pipelines[name]
		if !pipeline.Serverless || len(pipeline.Clusters) == 0 {
			continue
		}
		path := dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(name), dyn.Key("clusters"))
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "Cannot configure clusters for a serverless pipeline",
			Detail:    "Remove the clusters setting or set serverless to false.",
			Paths:     []dyn.Path{path},
			Locations: b.Config.GetLocations(path.String()),
		})
	}
	return diags
}
