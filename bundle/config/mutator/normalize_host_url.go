package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type normalizeHostURL struct{}

// NormalizeHostURL extracts query parameters from the workspace host URL
// and strips them from the host. This allows users to paste SPOG URLs
// (e.g. https://host.databricks.com/?o=12345) directly into their bundle config.
//
// The primary extraction happens in [config.Workspace.Client] before the SDK
// config is built. This mutator serves as a secondary normalization pass to
// ensure the bundle config is clean for any later code that reads it directly.
func NormalizeHostURL() bundle.Mutator {
	return &normalizeHostURL{}
}

func (m *normalizeHostURL) Name() string {
	return "NormalizeHostURL"
}

func (m *normalizeHostURL) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	b.Config.Workspace.NormalizeHostURL()
	return nil
}
