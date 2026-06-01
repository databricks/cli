package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type selectResources struct{}

// SelectResources returns a mutator that resolves and validates the selectors in
// b.Select. Selectors may be "type.name" (e.g. "jobs.myjob") or just "name" if
// unique across all resource types. The mutator does not filter the config; callers
// are responsible for filtering (via the plan graph or a direct config filter).
// If b.Select is empty, this is a no-op.
func SelectResources() bundle.Mutator {
	return &selectResources{}
}

func (m *selectResources) Name() string {
	return "SelectResources"
}

func (m *selectResources) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Select) == 0 {
		return nil
	}

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

// FilterSelectedResources filters b.Config.Resources to only the resources in
// b.Select (exact match, no dependency expansion). Used for commands that don't
// compute a deployment plan (validate, summary, terraform).
func FilterSelectedResources() bundle.Mutator {
	return &filterSelectedResources{}
}

type filterSelectedResources struct{}

func (m *filterSelectedResources) Name() string {
	return "FilterSelectedResources"
}

func (m *filterSelectedResources) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Select) == 0 {
		return nil
	}
	keep := make(map[string]struct{}, len(b.Select))
	for _, key := range b.Select {
		keep[key] = struct{}{}
	}
	b.Config.Resources.FilterResources(keep)
	return nil
}
