package terraform

import (
	"github.com/databricks/bricks/bundle"
)

// Dir returns the Terraform working directory for a given bundle.
// The working directory is emphemeral and nested under the bundle's cache directory.
func Dir(b *bundle.Bundle) (string, error) {
	return b.CacheDir("terraform")
}
