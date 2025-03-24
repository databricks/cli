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
	ctx = MockRuntime(ctx, Environment{IsDbr: true, Version: "15.4"})

	// Expect a panic if the mock function is run twice.
	assert.Panics(t, func() {
		MockRuntime(ctx, Environment{IsDbr: true, Version: "15.4"})
	})
}

func TestContext_RunsOnRuntimePanics(t *testing.T) {
	ctx := context.Background()

	// Expect a panic if the detection is not run.
	assert.Panics(t, func() {
		RunsOnRuntime(ctx)
	})
}

func TestContext_RuntimeVersionPanics(t *testing.T) {
	ctx := context.Background()

	// Expect a panic if the detection is not run.
	assert.Panics(t, func() {
		RuntimeVersion(ctx)
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

func TestContext_RuntimeVersion(t *testing.T) {
	ctx := context.Background()

	// Run detection.
	ctx = DetectRuntime(ctx)

	// Expect no panic because detection has run.
	assert.NotPanics(t, func() {
		RuntimeVersion(ctx)
	})
}

func TestContext_RunsOnRuntimeWithMock(t *testing.T) {
	ctx := context.Background()
	assert.True(t, RunsOnRuntime(MockRuntime(ctx, Environment{IsDbr: true, Version: "15.4"})))
	assert.False(t, RunsOnRuntime(MockRuntime(ctx, Environment{})))
}

func TestContext_RuntimeVersionWithMock(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "15.4", RuntimeVersion(MockRuntime(ctx, Environment{IsDbr: true, Version: "15.4"})))
	assert.Empty(t, RuntimeVersion(MockRuntime(ctx, Environment{})))
}
