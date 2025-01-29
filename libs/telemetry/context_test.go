package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithDefaultLogger(t *testing.T) {
	ctx := context.Background()

	// No default logger set
	ctx1 := WithDefaultLogger(ctx)
	assert.Equal(t, &defaultLogger{}, ctx1.Value(telemetryLoggerKey))

	// Default logger already set
	assert.PanicsWithError(t, "default telemetry logger already set in the context: *telemetry.defaultLogger", func() {
		WithDefaultLogger(ctx1)
	})

	// Mock logger already set
	ctx2 := WithMockLogger(ctx)
	assert.NotPanics(t, func() {
		WithDefaultLogger(ctx2)
	})

	// Unexpected logger type
	type foobar struct{}
	ctx3 := context.WithValue(ctx, telemetryLoggerKey, &foobar{})
	assert.PanicsWithError(t, "unexpected telemetry logger type: *telemetry.foobar", func() {
		WithDefaultLogger(ctx3)
	})
}

func TestWithMockLogger(t *testing.T) {
	ctx := context.Background()

	// No logger set
	ctx1 := WithMockLogger(ctx)
	assert.Equal(t, &mockLogger{}, ctx1.Value(telemetryLoggerKey))

	// Logger already set
	assert.PanicsWithError(t, "telemetry logger already set in the context: *telemetry.mockLogger", func() {
		WithMockLogger(ctx1)
	})

	// Default logger already set
	ctx2 := WithDefaultLogger(ctx)
	assert.PanicsWithError(t, "telemetry logger already set in the context: *telemetry.defaultLogger", func() {
		WithMockLogger(ctx2)
	})
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()

	// No logger set
	assert.PanicsWithError(t, "telemetry logger not found in the context", func() {
		fromContext(ctx)
	})

	// Default logger set
	ctx1 := WithDefaultLogger(ctx)
	assert.Equal(t, &defaultLogger{}, fromContext(ctx1))

	// Mock logger set
	ctx2 := WithMockLogger(ctx)
	assert.Equal(t, &mockLogger{}, fromContext(ctx2))

	// Unexpected logger type
	type foobar struct{}
	ctx3 := context.WithValue(ctx, telemetryLoggerKey, &foobar{})
	assert.PanicsWithError(t, "unexpected telemetry logger type: *telemetry.foobar", func() {
		fromContext(ctx3)
	})
}
