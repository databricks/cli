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
	// If the "compute_id" key is not set, just skip
	if b.Config.Bundle.ComputeId == "" {
		return nil
	}

	var diags diag.Diagnostics

	// The "compute_id" key is set; rewrite it to "cluster_id".
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		computeId, err := dyn.Get(v, "bundle.compute_id")
		if err != nil {
			return v, err
		}

		if computeId.Kind() != dyn.KindInvalid {
			p := dyn.NewPath(dyn.Key("bundle"), dyn.Key("compute_id"))
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   "compute_id is deprecated, please use cluster_id instead",
				Locations: computeId.Locations(),
				Paths:     []dyn.Path{p},
			})

			nv, err := dyn.Set(v, "bundle.cluster_id", computeId)
			if err != nil {
				return dyn.InvalidValue, err
			}
			// Drop the "compute_id" key.
			return dyn.Walk(nv, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				switch len(p) {
				case 0, 1:
					return v, nil
				case 2:
					if p[1] == dyn.Key("compute_id") {
						return v, dyn.ErrDrop
					}
				}
				return v, dyn.ErrSkip
			})
		}

		return v, nil
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
