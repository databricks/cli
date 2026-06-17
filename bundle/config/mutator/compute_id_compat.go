package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
)

type computeIdToClusterId struct{}

func ComputeIdToClusterId() bundle.Mutator {
	return &computeIdToClusterId{}
}

func (m *computeIdToClusterId) Name() string {
	return "ComputeIdToClusterId"
}

func (m *computeIdToClusterId) Apply(ctx context.Context, b *bundle.Bundle) error {
	// The "compute_id" key is set; rewrite it to "cluster_id".
	return b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		v, err := rewriteComputeIdToClusterId(ctx, v, dyn.NewPath(dyn.Key("bundle")))
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Check if the "compute_id" key is set in any target overrides.
		return dyn.MapByPattern(v, dyn.NewPattern(dyn.Key("targets"), dyn.AnyKey()), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return rewriteComputeIdToClusterId(ctx, v, dyn.Path{})
		})
	})
}

func rewriteComputeIdToClusterId(ctx context.Context, v dyn.Value, p dyn.Path) (dyn.Value, error) {
	computeIdPath := p.Append(dyn.Key("compute_id"))
	computeId, err := dyn.GetByPath(v, computeIdPath)
	// If the "compute_id" key is not set, we don't need to do anything.
	if err != nil {
		return v, nil //nolint:nilerr // missing key is not an error here
	}

	if computeId.Kind() == dyn.KindInvalid {
		return v, nil
	}

	logdiag.LogDiag(ctx, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   "compute_id is deprecated, please use cluster_id instead",
		Locations: computeId.Locations(),
		Paths:     []dyn.Path{computeIdPath},
	})

	clusterIdPath := p.Append(dyn.Key("cluster_id"))
	nv, err := dyn.SetByPath(v, clusterIdPath, computeId)
	if err != nil {
		return dyn.InvalidValue, err
	}
	// Drop the "compute_id" key.
	return dyn.Walk(nv, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
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
}
