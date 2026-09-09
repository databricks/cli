package logdiag_test

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/safeerr"
	"github.com/stretchr/testify/assert"
)

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

// TestGetFirstErrorSafe records that the first error's safe form is captured
// alongside its summary, and that a later error does not displace it.
func TestGetFirstErrorSafe(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	assert.Empty(t, logdiag.GetFirstErrorSafe(ctx))

	logdiag.LogError(ctx, safeerr.Errorf("cannot reach %s", safeerr.Safe("jobs.*")))
	logdiag.LogError(ctx, safeerr.New("second"))

	assert.Equal(t, "cannot reach jobs.*", logdiag.GetFirstErrorSafe(ctx))
}
