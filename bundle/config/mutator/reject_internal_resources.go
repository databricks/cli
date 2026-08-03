package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type rejectInternalResources struct{}

// RejectInternalResources returns a mutator that errors when a user has set
// any internal resource field in their bundle configuration.
func RejectInternalResources() bundle.Mutator {
	return &rejectInternalResources{}
}

func (m *rejectInternalResources) Name() string {
	return "RejectInternalResources"
}

func (m *rejectInternalResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	if b.Config.Resources.HasInternalResources() {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Internal resources cannot be set in bundle configuration",
			Paths:    []dyn.Path{dyn.MustPathFromString("resources")},
		})
	}

	return diags
}
