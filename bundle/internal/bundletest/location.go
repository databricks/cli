package bundletest

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/config"
)

// SetLocation sets the location of all values in the bundle to the given path.
// This is useful for testing where we need to associate configuration
// with the path it is loaded from.
func SetLocation(b *bundle.Bundle, pathPrefix config.Path, filePath string) {
	b.Config.Mutate(func(root config.Value) (config.Value, error) {
		return config.Walk(root, func(p config.Path, v config.Value) (config.Value, error) {
			// If the path has the given prefix, set the location.
			if p.HasPrefix(pathPrefix) {
				return v.WithLocation(config.Location{
					File: filePath,
				}), nil
			}

			// The path is not nested under the given prefix.
			// If the path is a prefix of the prefix, keep traversing and return the node verbatim.
			if pathPrefix.HasPrefix(p) {
				return v, nil
			}

			// Return verbatim, but skip traversal.
			return v, config.ErrSkip
		})
	})

	b.Config.ConfigureConfigFilePath()
}
