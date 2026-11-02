package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type enum struct{}

func Enum() bundle.Mutator {
	return &enum{}
}

func (f *enum) Name() string {
	return "validate:enum"
}

// Apply reports enum fields with invalid values. The walker is generated
// (enum_walk.go) by bundle/internal/validation.
func (f *enum) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := walkConfig(&b.Config, false, enumConfigRoot)
	sortDiagnostics(diags)
	return diags
}
