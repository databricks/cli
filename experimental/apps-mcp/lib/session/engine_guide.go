package session

import (
	"context"
	_ "embed"

	"github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
)

//go:embed initialization_message.md
var initializationMessageText string

// InitializationMessage is the initialization message injected on first tool call.
var InitializationMessage = initializationMessageText

// TrajectoryTracker interface to avoid import cycle
type TrajectoryTracker interface {
	RecordToolCall(toolName string, args any, result *mcp.CallToolResult, err error)
}

// WrapToolHandler wraps a tool handler to inject the ENGINE_GUIDE on first tool call
// and record trajectory if enabled
func WrapToolHandler[T any](session *Session, handler func(ctx context.Context, req *mcp.CallToolRequest, args T) (*mcp.CallToolResult, any, error)) func(ctx context.Context, req *mcp.CallToolRequest, args T) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args T) (*mcp.CallToolResult, any, error) {
		// Check if this is the first tool call
		isFirst := session.IsFirstTool()

		// Execute the original handler
		result, data, err := handler(ctx, req, args)

		// Record trajectory if tracker is available
		if session.Tracker != nil {
			if tracker, ok := session.Tracker.(TrajectoryTracker); ok {
				tracker.RecordToolCall(req.Params.Name, args, result, err)
			}
		}

		// If this was the first call and execution was successful, prepend the guide
		if err == nil && isFirst && result != nil && len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				textContent.Text = InitializationMessage + "\n\n---\n\n" + textContent.Text
			}
		}

		return result, data, err
	}
}
