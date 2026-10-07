package resourcemutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type mergeApps struct{}

func MergeApps() bundle.Mutator {
	return &mergeApps{}
}

func (m *mergeApps) Name() string {
	return "MergeApps"
}

func (m *mergeApps) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	names := make([]string, 0, len(b.Config.Resources.Apps))
	for name := range b.Config.Resources.Apps {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		app := b.Config.Resources.Apps[name]
		if app == nil {
			continue
		}
		path := structpath.NewPath(nil, "resources", "apps", name, "resources")
		if keyedMergeIsNoop(b, "resources.apps."+name+".resources", app.Resources, func(r apps.AppResource) string { return r.Name }, false) {
			continue
		}

		// Elements with the same name are overridden by the later ones. The result
		// is ordered by the first appearance of the name.
		var order []string
		last := map[string]int{}
		for i, r := range app.Resources {
			if _, ok := last[r.Name]; !ok {
				order = append(order, r.Name)
			}
			last[r.Name] = i
		}

		merged := make([]apps.AppResource, 0, len(order))
		sources := make([][]int, 0, len(order))
		for _, key := range order {
			merged = append(merged, app.Resources[last[key]])
			sources = append(sources, []int{last[key]})
		}
		app.Resources = merged
		b.Config.UpdateSequence(path, sources)

		// The merge sets the key of every element, even if it is empty.
		for i, r := range merged {
			if r.Name != "" {
				continue
			}
			if err := b.Config.Set(structpath.NewPath(path, i, "name"), ""); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	return nil
}
