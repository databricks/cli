package telemetry

import (
	"context"
)

// Private type to store the telemetry logger in the context
type telemetryLogger int

// Key to store the telemetry logger in the context
var telemetryLoggerKey telemetryLogger

func NewContext(ctx context.Context) context.Context {
	_, ok := ctx.Value(telemetryLoggerKey).(*logger)
	if ok {
		panic("telemetry logger already exists in the context")
	}

	return context.WithValue(ctx, telemetryLoggerKey, &logger{protoLogs: []string{}})
}

func fromContext(ctx context.Context) *logger {
	l, ok := ctx.Value(telemetryLoggerKey).(*logger)
	if !ok {
		panic("telemetry logger not found in the context")
	}
	return l
}
