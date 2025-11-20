package middlewares

import (
	"github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/experimental/apps-mcp/lib/trajectory"
)

// NewTrajectoryMiddleware creates middleware that records tool calls in a trajectory tracker.
func NewTrajectoryMiddleware(tracker *trajectory.Tracker) mcp.Middleware {
	return mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		result, err := next()
		if tracker != nil {
			tracker.RecordToolCall(ctx.Request.Params.Name, ctx.Request.Params.Arguments, result, err)
		}

		return result, err
	})
}

// NewToolCounterMiddleware creates middleware that increments the tool call counter.
func NewToolCounterMiddleware(session *session.Session) mcp.Middleware {
	return mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		session.IncrementToolCalls()
		return next()
	})
}
