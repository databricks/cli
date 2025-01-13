package telemetry

import (
	"context"

	"github.com/databricks/cli/libs/telemetry/events"
)

type mockLogger struct {
	events []DatabricksCliLog
}

func (l *mockLogger) Log(_ context.Context, event DatabricksCliLog) {
	if l.events == nil {
		l.events = make([]DatabricksCliLog, 0)
	}
	l.events = append(l.events, event)
}

func (l *mockLogger) Flush(ctx context.Context, executionContext *events.ExecutionContext, apiClient DatabricksApiClient) {
	// Do nothing
}

func (l *mockLogger) Introspect() []DatabricksCliLog {
	return l.events
}
