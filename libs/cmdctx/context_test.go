package cmdctx

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCommandGenerateExecIdPanics(t *testing.T) {
	ctx := context.Background()

	// Set the execution ID.
	ctx = GenerateExecId(ctx)

	// Expect a panic if the execution ID is set twice.
	assert.Panics(t, func() {
		ctx = GenerateExecId(ctx)
	})
}

func TestCommandExecIdPanics(t *testing.T) {
	ctx := context.Background()

	// Expect a panic if the execution ID is not set.
	assert.Panics(t, func() {
		ExecId(ctx)
	})
}

func TestCommandGenerateExecId(t *testing.T) {
	ctx := context.Background()

	// Set the execution ID.
	ctx = GenerateExecId(ctx)

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
