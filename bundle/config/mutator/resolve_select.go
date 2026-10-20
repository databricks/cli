package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/diag"
)

type resolveSelect struct{}

// ResolveSelect returns a mutator that resolves and validates the selectors in
// b.Select, normalizing each to its qualified "type.name" form. Selectors may be
// "type.name" (e.g. "jobs.myjob") or just "name" if unique across all resource
// types. The mutator does not filter the config; the direct engine selects against
// the resolved keys later via plan.FilterToSelected.
// If b.Select is empty, this is a no-op.
func ResolveSelect() bundle.Mutator {
	return &resolveSelect{}
}

func (m *resolveSelect) Name() string {
	return "ResolveSelect"
}

func (m *resolveSelect) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Select) == 0 {
		return nil
	}

	b.Metrics.SetBoolValue(metrics.SelectUsed, true)

	resolved := make([]string, 0, len(b.Select))
	for _, selector := range b.Select {
		res, key, err := b.Config.Resources.ResolveKey(selector)
		if err != nil {
			return diag.FromErr(err)
		}
		resolved = append(resolved, res.ResourceDescription().PluralName+"."+key)
	}

	b.Select = resolved
	return nil
}
