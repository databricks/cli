package telemetry

import "context"

// TODO CONTINUE:
// 1. Continue cleaning up the telemetry PR. Cleanup the interfaces
// and add a good mock / testing support by storing this in the context.
//
// 2. Test the logging is being done correctly. All componets work fine.
//
// 3. Ask once more for review. Also announce plans to do this by separately
// spawning a new process. We can add a new CLI command in the executable to
// do so.
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
