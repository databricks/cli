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
