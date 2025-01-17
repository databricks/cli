package mutator

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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

func computeRelativeLocations(base string, v dyn.Value) []dyn.Location {
	// Skip values that don't have locations.
	// Examples include defaults or values that are set by the program itself.
	locs := v.Locations()
	if len(locs) == 0 {
		return nil
	}

	// Convert absolute paths to relative paths.
	for i := range locs {
		rel, err := filepath.Rel(base, locs[i].File)
		if err != nil {
			return nil
		}
		// Convert the path separator to forward slashes.
		// This makes it possible to compare output across platforms.
		locs[i].File = filepath.ToSlash(rel)
	}

	return locs
}

func (m *populateLocations) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	loc := make(map[string][]dyn.Location)
	_, err := dyn.Walk(b.Config.Value(), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Skip the root value.
		if len(p) == 0 {
			return v, nil
		}

		// Skip values that don't have locations.
		// Examples include defaults or values that are set by the program itself.
		locs := computeRelativeLocations(b.BundleRootPath, v)
		if len(locs) > 0 {
			// Semantics for a value having multiple locations can be found in [merge.Merge].
			// We don't need to externalize these at the moment, so we limit the number
			// of locations to 1 while still using a slice for the output. This allows us
			// to include multiple entries in the future if we need to.
			loc[p.String()] = locs[0:1]
		}

		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	b.Config.Locations = loc
	return nil
}
