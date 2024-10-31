package root

import (
	"context"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/google/uuid"
)

func withCommandExecIdInUserAgent(ctx context.Context) context.Context {
	// A UUID that'll will allow use to correlate multiple API requests made by
	// the same command invocation.
	// When we add telemetry to the CLI, this exec ID will allow allow us to
	// correlate logs in HTTP access logs with logs in Frontend logs.
	return useragent.InContext(ctx, "cmd-exec-id", uuid.New().String())
}
