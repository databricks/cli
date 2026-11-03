package validate

import (
	"cmp"
	"context"
	"fmt"
	"slices"

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

// Apply reports enum fields with invalid values. The walker is generated (enum_walk.go)
// by bundle/internal/validation.
func (f *enum) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	w := &walker{diags: diag.Diagnostics{}}
	n := rootNode(b.Config.Value())
	enumConfigRoot(w, &n, &b.Config)
	diags := w.diags

	// Start from the order of a walk of the configuration tree, so that the sort
	// below breaks ties between diagnostics as it did with that walk.
	sortByWalkOrder(b.Config.Value(), diags)

	// Sort diagnostics to make them deterministic
	slices.SortFunc(diags, func(a, b diag.Diagnostic) int {
		// First sort by summary
		if n := cmp.Compare(a.Summary, b.Summary); n != 0 {
			return n
		}

		// Then sort by locations as a tie breaker if summaries are the same.
		return cmp.Compare(fmt.Sprintf("%v", a.Locations), fmt.Sprintf("%v", b.Locations))
	})

	return diags
}
