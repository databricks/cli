package bundletest

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
)

// SetLocation sets the location of all values in the bundle to the given path.
// This is useful for testing where we need to associate configuration
// with the path it is loaded from.
func SetLocation(b *bundle.Bundle, prefix string, locations []dyn.Location) {
	start := dyn.MustPathFromString(prefix)
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// If the path has the given prefix, set the location.
			if p.HasPrefix(start) {
				return v.WithLocations(locations), nil
			}

			// The path is not nested under the given prefix.
			// If the path is a prefix of the prefix, keep traversing and return the node verbatim.
			if start.HasPrefix(p) {
				return v, nil
			}

			// Return verbatim, but skip traversal.
			return v, dyn.ErrSkip
		})
	})
	if err != nil {
		panic("Mutate() failed: " + err.Error())
	}
}
