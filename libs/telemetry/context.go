package telemetry

import (
	"context"
	"fmt"
)

// Private type to store the telemetry logger in the context
type telemetryLogger int

// Key to store the telemetry logger in the context
var telemetryLoggerKey telemetryLogger

func WithDefaultLogger(ctx context.Context) context.Context {
	v := ctx.Value(telemetryLoggerKey)

	// If no logger is set in the context, set the default logger.
	if v == nil {
		nctx := context.WithValue(ctx, telemetryLoggerKey, &defaultLogger{})
		return nctx
	}

	switch v.(type) {
	case *defaultLogger:
		panic(fmt.Errorf("default telemetry logger already set in the context: %T", v))
	case *mockLogger:
		// Do nothing. Unit and integration tests set the mock logger in the context
		// to avoid making actual API calls. Thus WithDefaultLogger should silently
		// ignore the mock logger.
	default:
		panic(fmt.Errorf("unexpected telemetry logger type: %T", v))
	}

	return ctx
}

// WithMockLogger sets a mock telemetry logger in the context. It overrides the
// default logger if it is already set in the context.
func WithMockLogger(ctx context.Context) context.Context {
	v := ctx.Value(telemetryLoggerKey)
	if v != nil {
		panic(fmt.Errorf("telemetry logger already set in the context: %T", v))
	}

	return context.WithValue(ctx, telemetryLoggerKey, &mockLogger{})
}

func fromContext(ctx context.Context) Logger {
	v := ctx.Value(telemetryLoggerKey)
	if v == nil {
		panic(fmt.Errorf("telemetry logger not found in the context"))
	}

	switch vv := v.(type) {
	case *defaultLogger:
		return vv
	case *mockLogger:
		return vv
	default:
		panic(fmt.Errorf("unexpected telemetry logger type: %T", v))
	}
}
