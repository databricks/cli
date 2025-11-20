package middlewares

import (
	_ "embed"

	"github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
)

//go:embed initialization_message.md
var initializationMessageText string

// InitializationMessage is the initialization message injected on first tool call.
var InitializationMessage = initializationMessageText

// NewEngineGuideMiddleware creates middleware that injects the initialization message on the first tool call.
func NewEngineGuideMiddleware() mcp.Middleware {
	return mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		isFirst, ok := ctx.SessionData.GetBool("isFirstToolCall")
		if !ok {
			isFirst = true
		}

		// If this was the first call and execution was successful, prepend the guide
		if isFirst {
			ctx.SessionData.Set("isFirstToolCall", false)
			result, err := next()

			if err == nil && result != nil && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
					textContent.Text = InitializationMessage + "\n\n---\n\n" + textContent.Text
				}
			}

			return result, err
		}
		return next()
	})
}
