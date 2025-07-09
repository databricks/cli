package terraform

import (
	"context"

	"github.com/databricks/cli/bundle"
)

// Dir returns the Terraform working directory for a given bundle.
// The working directory is emphemeral and nested under the bundle's cache directory.
func Dir(ctx context.Context, b *bundle.Bundle) (string, error) {
	return b.LocalStateDir(ctx, "terraform")
}
