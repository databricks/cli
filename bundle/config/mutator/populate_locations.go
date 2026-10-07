package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
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

// locationPatterns are the paths for which locations are recorded.
var locationPatterns = []*structpath.PatternNode{
	structpath.MustParsePattern("*"),                         // Top level fields
	structpath.MustParsePattern("resources.*"),               // Resource groups ("resources.jobs")
	structpath.MustParsePattern("resources.*.*"),             // Resources for all types ("resources.jobs.my_job")
	structpath.MustParsePattern("resources.jobs.*.tasks"),    // Job tasks ("resources.jobs.my_job.tasks")
	structpath.MustParsePattern("resources.jobs.*.tasks[*]"), // Job task items ("resources.jobs.my_job.tasks[2]")
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

func gatherLocations(v structvar.View) (map[string][]diag.Location, error) {
	locs := map[string][]diag.Location{}
	for _, pattern := range locationPatterns {
		err := structvar.ForEach(v, pattern, func(p *structpath.PathNode, v structvar.View) error {
			locs[pathString(p)] = v.Locations()
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return locs, nil
}

func (m *populateLocations) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	pathToLocations, err := gatherLocations(b.Config.View())
	if err != nil {
		return diag.FromErr(err)
	}

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
