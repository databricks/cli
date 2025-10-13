// Copied from cmd/root/user_agent_command_exec_id.go and adapted for pipelines use.
package root

import (
	"context"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/useragent"
)

func withCommandExecIdInUserAgent(ctx context.Context) context.Context {
	// A UUID that will allow us to correlate multiple API requests made by
	// the same CLI invocation.
	return useragent.InContext(ctx, "cmd-exec-id", cmdctx.ExecId(ctx))
}
