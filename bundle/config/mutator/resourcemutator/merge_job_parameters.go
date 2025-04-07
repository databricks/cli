package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type mergeJobParameters struct{}

func MergeJobParameters() bundle.Mutator {
	return &mergeJobParameters{}
}

func (m *mergeJobParameters) Name() string {
	return "MergeJobParameters"
}

func (m *mergeJobParameters) parameterNameString(v dyn.Value) string {
	switch v.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		return ""
	case dyn.KindString:
		return v.MustString()
	default:
		panic("task key must be a string")
	}
}

func (m *mergeJobParameters) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v.Kind() == dyn.KindNil {
			return v, nil
		}

		return dyn.Map(v, "resources.jobs", dyn.Foreach(func(_ dyn.Path, job dyn.Value) (dyn.Value, error) {
			return dyn.Map(job, "parameters", merge.ElementsByKey("name", m.parameterNameString))
		}))
	})

	return diag.FromErr(err)
}
