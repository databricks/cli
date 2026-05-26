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
		// Validated in Apply before this is reached; unreachable under normal operation.
		return ""
	}
}

// validateResourceNames walks resources.app_spaces.*.resources[*].name and returns
// diagnostics for any entries where the name is not a string.
func (m *mergeAppSpaces) validateResourceNames(root dyn.Value) diag.Diagnostics {
	var diags diag.Diagnostics

	spaces := root.Get("resources").Get("app_spaces")
	if spaces.Kind() != dyn.KindMap {
		return nil
	}

	for _, spaceKV := range spaces.MustMap().Pairs() {
		resources := spaceKV.Value.Get("resources")
		if resources.Kind() != dyn.KindSequence {
			continue
		}
		for _, r := range resources.MustSequence() {
			name := r.Get("name")
			switch name.Kind() {
			case dyn.KindInvalid, dyn.KindNil, dyn.KindString:
				continue
			default:
				diags = diags.Extend(diag.Diagnostics{{
					Summary:   "app space resource name must be a string",
					Locations: []dyn.Location{name.Location()},
					Severity:  diag.Error,
				}})
			}
		}
	}

	return diags
}

func (m *mergeAppSpaces) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if diags := m.validateResourceNames(b.Config.Value()); diags.HasError() {
		return diags
	}

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
