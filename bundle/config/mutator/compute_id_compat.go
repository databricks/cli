package mutator

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
	view := b.Config.View()
	bundlePath := structpath.NewStringKey(nil, "bundle")
	diags = diags.Extend(rewriteComputeIdToClusterId(b, bundlePath, view.Get("bundle"), bundlePath))

	// Check if the "compute_id" key is set in any target overrides.
	err := structvar.ForEach(view, structpath.MustParsePattern("targets.*"), func(p *structpath.PathNode, v structvar.View) error {
		diags = diags.Extend(rewriteComputeIdToClusterId(b, p, v, nil))
		return nil
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}

// rewriteComputeIdToClusterId rewrites the "compute_id" key of the map v at path p to "cluster_id".
// The diagnostic refers to "compute_id" relative to diagPath.
func rewriteComputeIdToClusterId(b *bundle.Bundle, p *structpath.PathNode, v structvar.View, diagPath *structpath.PathNode) diag.Diagnostics {
	var diags diag.Diagnostics
	computeId := v.Get("compute_id")
	// If the "compute_id" key is not set, we don't need to do anything.
	if !computeId.IsValid() {
		return nil
	}

	diags = diags.Append(diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   "compute_id is deprecated, please use cluster_id instead",
		Locations: computeId.Locations(),
		Paths:     []*structpath.PathNode{structpath.NewStringKey(diagPath, "compute_id")},
	})

	err := b.Config.Assign(structpath.NewStringKey(p, "cluster_id"), computeId)
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}
	// Drop the "compute_id" key.
	return diags.Extend(diag.FromErr(b.Config.Delete(structpath.NewStringKey(p, "compute_id"))))
}
