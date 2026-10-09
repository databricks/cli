package bundletest

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

// SetLocation sets the location of all values in the bundle to the given path.
// This is useful for testing where we need to associate configuration
// with the path it is loaded from.
func SetLocation(b *bundle.Bundle, prefix string, locations []diag.Location) {
	// "." is the root of the configuration.
	var path *structpath.PathNode
	if prefix != "." {
		path = structpath.MustParsePath(prefix)
	}
	b.Config.SetLocations(path, locations)
}
