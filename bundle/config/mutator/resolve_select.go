package mutator

import (
	"context"
	"strings"

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

func (m *resolveSelect) Apply(_ context.Context, b *bundle.Bundle) error {
	if len(b.Select) == 0 {
		return nil
	}

	b.Metrics.SetBoolValue(metrics.SelectUsed, true)

	// Build reverse index: unqualified name → []"type.name" matches.
	byName := map[string][]string{}
	for _, group := range b.Config.Resources.AllResources() {
		typeName := group.Description.PluralName
		for name := range group.Resources {
			byName[name] = append(byName[name], typeName+"."+name)
		}
	}

	resolved := make([]string, 0, len(b.Select))
	for _, selector := range b.Select {
		if strings.Contains(selector, ".") {
			typeName, name, _ := strings.Cut(selector, ".")
			found := false
			for _, group := range b.Config.Resources.AllResources() {
				if group.Description.PluralName == typeName {
					if _, ok := group.Resources[name]; ok {
						found = true
					}
					break
				}
			}
			if !found {
				return diag.Errorf("no such resource: %s", selector)
			}
			resolved = append(resolved, selector)
		} else {
			matches := byName[selector]
			switch len(matches) {
			case 0:
				return diag.Errorf("no such resource: %s", selector)
			case 1:
				resolved = append(resolved, matches[0])
			default:
				return diag.Errorf("ambiguous resource: %s (can resolve to %s); use a qualified name to disambiguate", selector, strings.Join(matches, ", "))
			}
		}
	}

	b.Select = resolved
	return nil
}
