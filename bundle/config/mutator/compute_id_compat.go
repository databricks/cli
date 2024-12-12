package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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

	// The "compute_id" key is set; rewrite it to "cluster_id".
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		v, d := rewriteComputeIdToClusterId(v, dyn.NewPath(dyn.Key("bundle")))
		diags = diags.Extend(d)

		// Check if the "compute_id" key is set in any target overrides.
		return dyn.MapByPattern(v, dyn.NewPattern(dyn.Key("targets"), dyn.AnyKey()), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			v, d := rewriteComputeIdToClusterId(v, dyn.Path{})
			diags = diags.Extend(d)
			return v, nil
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}

func rewriteComputeIdToClusterId(v dyn.Value, p dyn.Path) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	computeIdPath := p.Append(dyn.Key("compute_id"))
	computeId, err := dyn.GetByPath(v, computeIdPath)
	// If the "compute_id" key is not set, we don't need to do anything.
	if err != nil {
		return v, nil
	}

	if computeId.Kind() == dyn.KindInvalid {
		return v, nil
	}

	diags = diags.Append(diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   "compute_id is deprecated, please use cluster_id instead",
		Locations: computeId.Locations(),
		Paths:     []dyn.Path{computeIdPath},
	})

	clusterIdPath := p.Append(dyn.Key("cluster_id"))
	nv, err := dyn.SetByPath(v, clusterIdPath, computeId)
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(err)
	}
	// Drop the "compute_id" key.
	vout, err := dyn.Walk(nv, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		switch len(p) {
		case 0:
			return v, nil
		case 1:
			if p[0] == dyn.Key("compute_id") {
				return v, dyn.ErrDrop
			}
			return v, nil
		case 2:
			if p[1] == dyn.Key("compute_id") {
				return v, dyn.ErrDrop
			}
		}
		return v, dyn.ErrSkip
	})

	diags = diags.Extend(diag.FromErr(err))
	return vout, diags
}
