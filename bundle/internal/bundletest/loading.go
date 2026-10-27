package bundletest

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/require"
)

// ApplyLoading applies mutators of the load phase, which work on the configuration
// tree, to b (whose configuration may be typed) and converts the result into the
// typed configuration, like the load phase does.
func ApplyLoading(t testing.TB, b *bundle.Bundle, mutators ...bundle.Mutator) diag.Diagnostics {
	require.NoError(t, b.Config.StartLoading())
	diags := bundle.ApplySeq(t.Context(), b, mutators...)
	finishDiags, err := b.Config.FinishLoading()
	require.NoError(t, err)
	return diags.Extend(finishDiags)
}
