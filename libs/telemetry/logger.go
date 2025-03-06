package telemetry

import (
	"context"

	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/google/uuid"
)

func Log(ctx context.Context, event protos.DatabricksCliLog) {
	fromContext(ctx).log(event)
}

func GetLogs(ctx context.Context) []protos.FrontendLog {
	return fromContext(ctx).getLogs()
}

func HasLogs(ctx context.Context) bool {
	return len(fromContext(ctx).getLogs()) > 0
}

func SetExecutionContext(ctx context.Context, ec protos.ExecutionContext) {
	fromContext(ctx).setExecutionContext(ec)
}

type logger struct {
	logs []protos.FrontendLog
}

func (l *logger) log(event protos.DatabricksCliLog) {
	if l.logs == nil {
		l.logs = make([]protos.FrontendLog, 0)
	}
	l.logs = append(l.logs, protos.FrontendLog{
		FrontendLogEventID: uuid.New().String(),
		Entry: protos.FrontendLogEntry{
			DatabricksCliLog: event,
		},
	})
}

func (l *logger) getLogs() []protos.FrontendLog {
	return l.logs
}

func (l *logger) setExecutionContext(ec protos.ExecutionContext) {
	for i := range l.logs {
		l.logs[i].Entry.DatabricksCliLog.ExecutionContext = &ec
	}
}
