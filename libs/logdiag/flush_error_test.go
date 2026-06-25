package logdiag

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlushErrorIdempotent(t *testing.T) {
	ctx := InitContext(t.Context())
	SetCollect(ctx, true)

	flushed := FlushError(ctx, errors.New("boom"))
	require.Error(t, flushed)
	// The wrapped error matches the sentinel but preserves the message.
	assert.ErrorIs(t, flushed, ErrAlreadyPrinted)
	assert.Equal(t, "boom", flushed.Error())

	// Flushing the already-flushed error is a no-op: it returns the same error
	// and does not render a second diagnostic.
	again := FlushError(ctx, flushed)
	assert.Equal(t, flushed, again)

	diags := FlushCollected(ctx)
	assert.Len(t, diags, 1)
	assert.Equal(t, "boom", diags[0].Summary)
}

func TestFlushErrorNil(t *testing.T) {
	ctx := InitContext(t.Context())
	SetCollect(ctx, true)

	assert.NoError(t, FlushError(ctx, nil))
	assert.Empty(t, FlushCollected(ctx))
}

func TestFlushErrorJoinRendersEachLeaf(t *testing.T) {
	ctx := InitContext(t.Context())
	SetCollect(ctx, true)

	_ = FlushError(ctx, errors.Join(errors.New("one"), errors.New("two")))

	diags := FlushCollected(ctx)
	require.Len(t, diags, 2)
	assert.Equal(t, "one", diags[0].Summary)
	assert.Equal(t, "two", diags[1].Summary)
}
