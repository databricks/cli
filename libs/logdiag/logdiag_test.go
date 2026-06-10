package logdiag_test

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
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
