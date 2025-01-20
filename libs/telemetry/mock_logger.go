package telemetry

import "context"

type mockLogger struct {
	events []DatabricksCliLog
}

func (l *mockLogger) Log(event DatabricksCliLog) {
	if l.events == nil {
		l.events = make([]DatabricksCliLog, 0)
	}
	l.events = append(l.events, event)
}

func (l *mockLogger) Flush(ctx context.Context, apiClient DatabricksApiClient) {
	// Do nothing
}

func (l *mockLogger) Introspect() []DatabricksCliLog {
	return l.events
}
