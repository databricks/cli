package dbr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext_DetectRuntimePanics(t *testing.T) {
	ctx := context.Background()

	// Run detection.
	ctx = DetectRuntime(ctx)

	// Expect a panic if the detection is run twice.
	assert.Panics(t, func() {
		ctx = DetectRuntime(ctx)
	})
}

func TestContext_MockRuntimePanics(t *testing.T) {
	ctx := context.Background()

	// Run detection.
	ctx = MockRuntime(ctx, true)

	// Expect a panic if the mock function is run twice.
	assert.Panics(t, func() {
		MockRuntime(ctx, true)
	})
}

func TestContext_RunsOnRuntimePanics(t *testing.T) {
	ctx := context.Background()

	// Expect a panic if the detection is not run.
	assert.Panics(t, func() {
		RunsOnRuntime(ctx)
	})
}

func TestContext_RunsOnRuntime(t *testing.T) {
	ctx := context.Background()

	// Run detection.
	ctx = DetectRuntime(ctx)

	// Expect no panic because detection has run.
	assert.NotPanics(t, func() {
		RunsOnRuntime(ctx)
	})
}

func TestContext_RunsOnRuntimeWithMock(t *testing.T) {
	ctx := context.Background()
	assert.True(t, RunsOnRuntime(MockRuntime(ctx, true)))
	assert.False(t, RunsOnRuntime(MockRuntime(ctx, false)))
}
