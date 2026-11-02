package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loctable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type populateLocations struct{}

// PopulateLocations collects location information for the entire configuration tree
// and includes this as the [config.Root.Locations] property.
func PopulateLocations() bundle.Mutator {
	return &populateLocations{}
}

func (m *populateLocations) Name() string {
	return "PopulateLocations"
}

// pathString returns the path with keys separated by dots, whatever characters they contain.
func pathString(p *structpath.PathNode) string {
	var b strings.Builder
	for i, n := range p.AsSlice() {
		if k, ok := n.StringKey(); ok {
			if i > 0 {
				b.WriteByte('.')
			}
			b.WriteString(k)
		} else if idx, ok := n.Index(); ok {
			fmt.Fprintf(&b, "[%d]", idx)
		}
	}
	return b.String()
}

// gatherLocations returns the locations of the top level fields, the resource groups
// ("resources.jobs"), the resources ("resources.jobs.my_job"), and job tasks
// ("resources.jobs.my_job.tasks", "resources.jobs.my_job.tasks[2]"). Resources and tasks
// come from the configuration, their locations from the locations tree (values without
// locations have no entry).
func gatherLocations(r *config.Root) map[string][]diag.Location {
	out := map[string][]diag.Location{}
	tree := r.LocationTree()
	for k, top := range tree.KeyItems() {
		addLocations(out, structpath.NewPath(nil, k), top)
	}
	groups := tree.Key("resources")
	for group, g := range groups.KeyItems() {
		addLocations(out, structpath.NewPath(nil, "resources", group), g)
	}
	r.Resources.IterateAllResources(func(group, name string, _ config.ConfigResource) bool {
		rp := structpath.NewPath(nil, "resources", group, name)
		res := groups.Key(group).Key(name)
		addLocations(out, rp, res)
		if group == "jobs" {
			if job := r.Resources.Jobs[name]; job != nil {
				tasks := res.Key("tasks")
				addLocations(out, structpath.NewPath(rp, "tasks"), tasks)
				for i := range job.Tasks {
					addLocations(out, structpath.NewPath(rp, "tasks", i), tasks.Index(i))
				}
			}
		}
		return true
	})
	return out
}

// addLocations records the locations of the value at p (from its tree l), if any.
func addLocations(out map[string][]diag.Location, p *structpath.PathNode, l *structvar.Locations) {
	if locs := l.Get(); len(locs) > 0 {
		out[pathString(p)] = locs
	}
}

func (m *populateLocations) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	pathToLocations := gatherLocations(&b.Config)

	locs, err := loctable.Build(
		pathToLocations,
		// Make all paths relative to the bundle root.
		b.BundleRootPath,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	b.Config.Locations = &locs
	return nil
}
