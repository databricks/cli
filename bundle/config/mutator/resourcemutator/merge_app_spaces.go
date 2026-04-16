package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type mergeAppSpaces struct{}

func MergeAppSpaces() bundle.Mutator {
	return &mergeAppSpaces{}
}

func (m *mergeAppSpaces) Name() string {
	return "MergeAppSpaces"
}

func (m *mergeAppSpaces) resourceName(v dyn.Value) string {
	switch v.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		return ""
	case dyn.KindString:
		return v.MustString()
	default:
		panic("app space resource name must be a string")
	}
}

func (m *mergeAppSpaces) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v.Kind() == dyn.KindNil {
			return v, nil
		}

		return dyn.Map(v, "resources.app_spaces", dyn.Foreach(func(_ dyn.Path, space dyn.Value) (dyn.Value, error) {
			return dyn.Map(space, "resources", merge.ElementsByKeyWithOverride("name", m.resourceName))
		}))
	})

	return diag.FromErr(err)
}
