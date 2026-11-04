package mutator

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type computeIdToClusterId struct{}

func ComputeIdToClusterId() bundle.Mutator {
	return &computeIdToClusterId{}
}

func (m *computeIdToClusterId) Name() string {
	return "ComputeIdToClusterId"
}

func (m *computeIdToClusterId) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Skip if "compute_id" is not set anywhere.
	if b.Config.Bundle.ComputeId == "" && !slices.ContainsFunc(slices.Collect(maps.Values(b.Config.Targets)), func(t *config.Target) bool {
		return t != nil && t.ComputeId != ""
	}) {
		return nil
	}

	// The "compute_id" key is set; rewrite it to "cluster_id".
	b2 := &b.Config.Bundle
	diags = diags.Extend(rewriteComputeIdToClusterId(b, structpath.NewStringKey(nil, "bundle"), &b2.ComputeId, &b2.ClusterId))

	// Check if the "compute_id" key is set in any target overrides.
	for _, name := range slices.Sorted(maps.Keys(b.Config.Targets)) {
		if t := b.Config.Targets[name]; t != nil {
			diags = diags.Extend(rewriteComputeIdToClusterId(b, structpath.NewPath(nil, "targets", name), &t.ComputeId, &t.ClusterId))
		}
	}
	return diags
}

// rewriteComputeIdToClusterId moves the compute_id of the value at p to its cluster_id,
// with its locations.
func rewriteComputeIdToClusterId(b *bundle.Bundle, p *structpath.PathNode, computeId, clusterId *string) diag.Diagnostics {
	// If the "compute_id" key is not set, we don't need to do anything.
	if *computeId == "" {
		return nil
	}

	computeIdPath := structpath.NewStringKey(p, "compute_id")
	locs := b.Config.GetLocationsOfExact(computeIdPath)
	diags := diag.Diagnostics{{
		Severity:  diag.Warning,
		Summary:   "compute_id is deprecated, please use cluster_id instead",
		Locations: locs,
		Paths:     []*structpath.PathNode{computeIdPath},
	}}

	*clusterId = *computeId
	b.Config.SetLocations(structpath.NewStringKey(p, "cluster_id"), locs)
	// Drop the "compute_id" key.
	return diags.Extend(diag.FromErr(b.Config.Delete(computeIdPath)))
}
