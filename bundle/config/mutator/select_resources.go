package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type selectResources struct{}

// SelectResources returns a mutator that filters bundle resources to those listed in b.Select.
// Selectors may be "type.name" (e.g. "jobs.myjob") or just "name" if unique across all resource types.
// Dependencies referenced via ${resources.*.*.*} are included transitively.
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

	// Build reverse index: unqualified name → []"type.name" matches
	byName := map[string][]string{}
	for _, group := range b.Config.Resources.AllResources() {
		typeName := group.Description.PluralName
		for name := range group.Resources {
			byName[name] = append(byName[name], typeName+"."+name)
		}
	}

	keep := map[string]struct{}{}
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
			keep[selector] = struct{}{}
		} else {
			matches := byName[selector]
			switch len(matches) {
			case 0:
				return diag.Errorf("no such resource: %s", selector)
			case 1:
				keep[matches[0]] = struct{}{}
			default:
				return diag.Errorf("ambiguous resource: %s (can resolve to %s); use a qualified name to disambiguate", selector, strings.Join(matches, ", "))
			}
		}
	}

	// Expand keep set transitively: for each kept resource, find resources it
	// references via unresolved ${resources.<type>.<name>.*} variables and add
	// them too. Repeat until no new resources are discovered.
	configVal := b.Config.Value()
	queue := make([]string, 0, len(keep))
	for key := range keep {
		queue = append(queue, key)
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		for _, dep := range resourceDeps(configVal, key) {
			if _, ok := keep[dep]; !ok {
				keep[dep] = struct{}{}
				queue = append(queue, dep)
			}
		}
	}

	b.Config.Resources.FilterResources(keep)
	return nil
}

// resourceDeps returns the "type.name" keys of resources referenced by unresolved
// ${resources.<type>.<name>.*} variables inside the given resource's config subtree.
func resourceDeps(root dyn.Value, key string) []string {
	path, err := dyn.NewPathFromString("resources." + key)
	if err != nil {
		return nil
	}
	val, err := dyn.GetByPath(root, path)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var deps []string
	_ = dyn.WalkReadOnly(val, func(_ dyn.Path, v dyn.Value) error {
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		for _, pathStr := range ref.References() {
			// pathStr is like "resources.jobs.bar.id"; extract "jobs.bar"
			parts := strings.SplitN(pathStr, ".", 4)
			if len(parts) < 3 || parts[0] != "resources" {
				continue
			}
			dep := parts[1] + "." + parts[2]
			if !seen[dep] {
				seen[dep] = true
				deps = append(deps, dep)
			}
		}
		return nil
	})
	return deps
}
