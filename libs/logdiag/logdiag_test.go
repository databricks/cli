package logdiag_test

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
)

func TestIsolatedContext(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	isolated := logdiag.IsolatedContext(ctx)
	logdiag.SetCollect(isolated, true)
	logdiag.LogDiag(isolated, diag.Diagnostic{Severity: diag.Error, Summary: "inner failure"})

	// The error is recorded in the isolated context only, not the parent.
	assert.Len(t, logdiag.FlushCollected(isolated), 1)
	assert.Empty(t, logdiag.FlushCollected(ctx))
}
