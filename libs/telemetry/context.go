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
	if v != nil {
		panic(fmt.Sprintf("telemetry logger already set in the context: %v", v))
	}

	return context.WithValue(ctx, telemetryLoggerKey, &defaultLogger{logs: []FrontendLog{}})
}

func WithMockLogger(ctx context.Context) context.Context {
	v := ctx.Value(telemetryLoggerKey)
	if v != nil {
		panic(fmt.Sprintf("telemetry logger already set in the context: %v", v))
	}

	return context.WithValue(ctx, telemetryLoggerKey, &mockLogger{})
}

func fromContext(ctx context.Context) Logger {
	v := ctx.Value(telemetryLoggerKey)
	if v == nil {
		panic("telemetry logger not found in the context")
	}

	switch v.(type) {
	case *defaultLogger:
		return v.(*defaultLogger)
	case *mockLogger:
		return v.(*mockLogger)
	default:
		panic(fmt.Sprintf("unexpected telemetry logger type: %T", v))
	}
}
