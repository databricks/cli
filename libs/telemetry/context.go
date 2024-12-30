package telemetry

import (
	"context"
)

// Private type to store the telemetry logger in the context
type telemetryLogger int

// Key to store the telemetry logger in the context
var telemetryLoggerKey telemetryLogger

func ContextWithLogger(ctx context.Context) context.Context {
	_, ok := ctx.Value(telemetryLoggerKey).(*logger)
	if ok {
		// If a logger is already configured in the context, do not set a new one.
		// This is useful for testing.
		return ctx
	}

	return context.WithValue(ctx, telemetryLoggerKey, &logger{logs: []FrontendLog{}})
}

func fromContext(ctx context.Context) *logger {
	l, ok := ctx.Value(telemetryLoggerKey).(*logger)
	if !ok {
		panic("telemetry logger not found in the context")
	}
	return l
}
