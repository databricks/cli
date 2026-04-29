package phases

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/stretchr/testify/assert"
)

// fakeDiagMutator emits a fixed diagnostics slice; used to exercise the
// warning→error promotion wrapper without depending on a real validator.
type fakeDiagMutator struct {
	diags diag.Diagnostics
}

func (f *fakeDiagMutator) Name() string { return "fakeDiagMutator" }

func (f *fakeDiagMutator) Apply(_ context.Context, _ *ucm.Ucm) diag.Diagnostics {
	return f.diags
}

func TestPromoteWarningsIfRequestedNoOpWhenDisabled(t *testing.T) {
	inner := &fakeDiagMutator{diags: diag.Diagnostics{
		{Severity: diag.Warning, Summary: "w"},
		{Severity: diag.Error, Summary: "e"},
		{Severity: diag.Recommendation, Summary: "r"},
	}}

	wrapped := promoteWarningsIfRequested(inner, false)

	out := wrapped.Apply(t.Context(), nil)
	assert.Equal(t, diag.Warning, out[0].Severity)
	assert.Equal(t, diag.Error, out[1].Severity)
	assert.Equal(t, diag.Recommendation, out[2].Severity)
}

func TestPromoteWarningsIfRequestedRewritesWarningSeverity(t *testing.T) {
	inner := &fakeDiagMutator{diags: diag.Diagnostics{
		{Severity: diag.Warning, Summary: "w"},
		{Severity: diag.Error, Summary: "e"},
		{Severity: diag.Recommendation, Summary: "r"},
	}}

	wrapped := promoteWarningsIfRequested(inner, true)
	assert.Equal(t, inner.Name(), wrapped.Name(), "wrapper must preserve Name() for telemetry")

	out := wrapped.Apply(t.Context(), nil)
	assert.Equal(t, diag.Error, out[0].Severity, "warning must be promoted to error")
	assert.Equal(t, diag.Error, out[1].Severity, "errors stay as errors")
	assert.Equal(t, diag.Recommendation, out[2].Severity, "recommendations are not promoted")
}
