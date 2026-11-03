package telemetry

import (
	"context"
	"errors"
)

// Private type to store the telemetry logger in the context
type telemetryLogger int

// Key to store the telemetry logger in the context
var telemetryLoggerKey telemetryLogger

func WithNewLogger(ctx context.Context) context.Context {
	return context.WithValue(ctx, telemetryLoggerKey, &logger{})
}

func fromContext(ctx context.Context) *logger {
	v := ctx.Value(telemetryLoggerKey)
	if v == nil {
		panic(errors.New("telemetry logger not found in the context"))
	}

	return v.(*logger)
}

// loggerFromContext returns the telemetry logger, or false if none was
// installed. Unlike fromContext it does not panic, so callers on paths that may
// run without telemetry setup can drop events instead of crashing.
func loggerFromContext(ctx context.Context) (*logger, bool) {
	v, ok := ctx.Value(telemetryLoggerKey).(*logger)
	return v, ok
}
