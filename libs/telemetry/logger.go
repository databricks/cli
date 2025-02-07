package telemetry

import (
	"context"

	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/google/uuid"
)

const SkipEnvVar = "DATABRICKS_CLI_SKIP_TELEMETRY"

// DATABRICKS_CLI_BLOCK_ON_TELEMETRY_UPLOAD is an environment variable that can be set
// to make the CLI process block until the telemetry logs are uploaded.
// Only used for testing.
const BlockOnUploadEnvVar = "DATABRICKS_CLI_BLOCK_ON_TELEMETRY_UPLOAD"

func Log(ctx context.Context, event protos.DatabricksCliLog) {
	fromContext(ctx).log(event)
}

func GetLogs(ctx context.Context) []protos.FrontendLog {
	return fromContext(ctx).getLogs()
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
