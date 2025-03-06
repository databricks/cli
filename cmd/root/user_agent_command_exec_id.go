package root

import (
	"context"

	"github.com/databricks/cli/libs/command"
	"github.com/databricks/databricks-sdk-go/useragent"
)

func withCommandExecIdInUserAgent(ctx context.Context) context.Context {
	return useragent.InContext(ctx, "cmd-exec-id", command.ExecId(ctx))
}
