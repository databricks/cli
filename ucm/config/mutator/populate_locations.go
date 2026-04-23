package mutator

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn/dynloc"
	"github.com/databricks/cli/ucm"
)

type populateLocations struct{}

// PopulateLocations collects location information for the entire configuration tree
// and includes this as the [config.Root.Locations] property.
func PopulateLocations() ucm.Mutator {
	return &populateLocations{}
}

func (m *populateLocations) Name() string {
	return "PopulateLocations"
}

func (m *populateLocations) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	locs, err := dynloc.Build(
		u.Config.Value(),
		// Make all paths relative to the ucm root.
		u.RootPath,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	u.Config.Locations = &locs
	return nil
}
