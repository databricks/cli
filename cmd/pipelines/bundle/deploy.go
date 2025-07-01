// Copied from cmd/bundle/deploy.go and adapted for pipelines use.
package bundle

import (
	"fmt"
	"io"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
)

// RenderDiagnostics renders the diagnostics in a human-readable format.
func RenderDiagnostics(w io.Writer, b *bundle.Bundle, diags diag.Diagnostics) error {
	renderOpts := render.RenderOptions{RenderSummaryTable: false}
	err := render.RenderDiagnostics(w, b, diags, renderOpts)
	if err != nil {
		return fmt.Errorf("failed to render output: %w", err)
	}

	if diags.HasError() {
		return root.ErrAlreadyPrinted
	}

	return nil
}
