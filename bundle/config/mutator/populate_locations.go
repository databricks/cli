package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn/dynloc"
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

func (m *populateLocations) Apply(ctx context.Context, b *bundle.Bundle) error {
	locs, err := dynloc.Build(
		b.Config.Value(),
		// Make all paths relative to the bundle root.
		b.BundleRootPath,
	)
	if err != nil {
		return err
	}

	b.Config.Locations = &locs
	return nil
}
