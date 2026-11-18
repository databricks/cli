package logdiag_test

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
)

func TestGetFirstErrorCause(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	assert.NoError(t, logdiag.GetFirstErrorCause(ctx))

	logdiag.LogDiag(ctx, diag.Diagnostic{Severity: diag.Warning, Summary: "warning", Cause: errors.New("warning cause")})
	first := errors.New("first")
	logdiag.LogError(ctx, first)
	logdiag.LogError(ctx, errors.New("second"))

	assert.Equal(t, first, logdiag.GetFirstErrorCause(ctx))
}

func TestGetFirstErrorCauseFromMessage(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	logdiag.LogDiag(ctx, diag.Errorf("made from a message")[0])
	logdiag.LogError(ctx, errors.New("later"))

	assert.NoError(t, logdiag.GetFirstErrorCause(ctx))
}

func TestIsolatedContext(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	isolated := logdiag.IsolatedContext(ctx)
	logdiag.SetCollect(isolated, true)
	logdiag.LogError(isolated, errors.New("inner failure"))

	assert.True(t, logdiag.HasError(isolated))
	assert.Len(t, logdiag.FlushCollected(isolated), 1)

	assert.False(t, logdiag.HasError(ctx))
	assert.Empty(t, logdiag.FlushCollected(ctx))
}
