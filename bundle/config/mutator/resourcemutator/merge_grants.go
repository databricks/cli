package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

// Resource types that support grants.
var grantResourceTypes = []string{
	"catalogs",
	"schemas",
	"external_locations",
	"volumes",
	"registered_models",
}

type mergeGrants struct{}

// MergeGrants returns a mutator that deduplicates grant entries.
// It merges entries with the same principal and deduplicates privileges.
func MergeGrants() bundle.Mutator {
	return &mergeGrants{}
}

func (m *mergeGrants) Name() string {
	return "MergeGrants"
}

func (m *mergeGrants) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v.Kind() == dyn.KindNil {
			return v, nil
		}

		for _, resourceType := range grantResourceTypes {
			var mapErr error
			v, mapErr = dyn.Map(v, "resources."+resourceType, dyn.Foreach(func(_ dyn.Path, resource dyn.Value) (dyn.Value, error) {
				// Merge grant entries by principal. This concatenates privileges
				// for entries with the same principal via the standard merge semantics.
				resource, err := dyn.Map(resource, "grants", merge.ElementsByKey("principal", func(v dyn.Value) string {
					s, _ := v.AsString()
					return s
				}))
				if err != nil {
					return resource, err
				}

				// Deduplicate privileges within each grant entry.
				return dyn.Map(resource, "grants", dyn.Foreach(func(_ dyn.Path, grant dyn.Value) (dyn.Value, error) {
					return dyn.Map(grant, "privileges", deduplicateSequence)
				}))
			}))
			if mapErr != nil {
				return v, mapErr
			}
		}

		return v, nil
	})

	return diag.FromErr(err)
}

// deduplicateSequence removes duplicate values from a dyn sequence,
// preserving the order of first appearance.
func deduplicateSequence(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
	elements, ok := v.AsSequence()
	if !ok {
		return v, nil
	}

	seen := make(map[string]bool, len(elements))
	out := make([]dyn.Value, 0, len(elements))
	for _, elem := range elements {
		key, _ := elem.AsString()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, elem)
	}

	return dyn.NewValue(out, v.Locations()), nil
}
