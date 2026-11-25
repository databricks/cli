package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type mergeApps struct{}

func MergeApps() bundle.Mutator {
	return &mergeApps{}
}

func (m *mergeApps) Name() string {
	return "MergeApps"
}

func (m *mergeApps) resourceName(v dyn.Value) string {
	switch v.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		return ""
	case dyn.KindString:
		return v.MustString()
	default:
		panic("app name must be a string")
	}
}

func (m *mergeApps) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v.Kind() == dyn.KindNil {
			return v, nil
		}

		return dyn.Map(v, "resources.apps", dyn.Foreach(func(_ dyn.Path, app dyn.Value) (dyn.Value, error) {
			return dyn.Map(app, "resources", merge.ElementsByKeyWithOverride("name", m.resourceName))
		}))
	})

	return diag.FromErr(err)
}
