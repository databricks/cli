package middlewares

import (
	"github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
)

// NewEngineGuideMiddleware creates middleware that injects the initialization message on the first tool call.
func NewEngineGuideMiddleware() mcp.Middleware {
	return mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		isFirst := ctx.Session.GetBool("isFirstToolCall", true)
		initializationMessage := prompts.MustExecuteTemplate("initialization_message.tmpl", nil)

		// If this was the first call and execution was successful, prepend the guide
		if isFirst {
			ctx.Session.Set("isFirstToolCall", false)
			result, err := next()
			if err != nil {
				result = mcp.CreateNewTextContentResultError(err)
			}
			if result != nil && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
					textContent.Text = initializationMessage + "\n\n---\n\n" + textContent.Text
				}
			}

			return result, nil
		}
		return next()
	})
}
