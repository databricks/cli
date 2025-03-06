package command

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContext_SetExecIdPanics(t *testing.T) {
	ctx := context.Background()

	// Set the execution ID.
	ctx = SetExecId(ctx)

	// Expect a panic if the execution ID is set twice.
	assert.Panics(t, func() {
		ctx = SetExecId(ctx)
	})
}

func TestContext_MockExecIdPanics(t *testing.T) {
	ctx := context.Background()

	// Set the execution ID.
	ctx = MockExecId(ctx, "test")

	// Expect a panic if the mock function is run twice.
	assert.Panics(t, func() {
		MockExecId(ctx, "test")
	})
}

func TestContext_ExecIdPanics(t *testing.T) {
	ctx := context.Background()

	// Expect a panic if the execution ID is not set.
	assert.Panics(t, func() {
		ExecId(ctx)
	})
}

func TestContext_ExecId(t *testing.T) {
	ctx := context.Background()

	// Set the execution ID.
	ctx = SetExecId(ctx)

	// Expect no panic because the execution ID is set.
	assert.NotPanics(t, func() {
		ExecId(ctx)
	})

	v := ExecId(ctx)

	// Subsequent calls should return the same value.
	assert.Equal(t, v, ExecId(ctx))

	// The value should be a valid UUID.
	assert.NoError(t, uuid.Validate(v))
}

func TestContext_ExecIdWithMock(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "test", ExecId(MockExecId(ctx, "test")))
	assert.Equal(t, "test2", ExecId(MockExecId(ctx, "test2")))
}
