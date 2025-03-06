package root

import (
	"context"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/google/uuid"
)

var cmdExecId = uuid.New().String()

func withCommandExecIdInUserAgent(ctx context.Context) context.Context {
	// A UUID that will allow us to correlate multiple API requests made by
	// the same CLI invocation.
	return useragent.InContext(ctx, "cmd-exec-id", cmdExecId)
}
