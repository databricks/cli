package mcp

import (
	"context"

	"github.com/databricks/cli/experimental/aitools/lib/session"
)

// MiddlewareContext provides context for middleware execution.
type MiddlewareContext struct {
	Ctx     context.Context
	Request *CallToolRequest
	Session *session.Session
}

// MiddlewareFunc is a function that processes a tool call request.
// It can:
// - Return (nil, nil) to pass execution to the next middleware or tool handler
// - Return (result, nil) to short-circuit and return a result immediately
// - Return (nil, error) to abort execution with an error
type MiddlewareFunc func(*MiddlewareContext, NextFunc) (*CallToolResult, error)

// NextFunc is called by middleware to pass execution to the next middleware or tool handler.
type NextFunc func() (*CallToolResult, error)

// Middleware represents a middleware component in the chain.
type Middleware interface {
	// Handle processes the request and optionally calls next to continue the chain.
	Handle(ctx *MiddlewareContext, next NextFunc) (*CallToolResult, error)
}

// MiddlewareFuncAdapter adapts a MiddlewareFunc to the Middleware interface.
type MiddlewareFuncAdapter struct {
	fn MiddlewareFunc
}

// Handle implements the Middleware interface.
func (m *MiddlewareFuncAdapter) Handle(ctx *MiddlewareContext, next NextFunc) (*CallToolResult, error) {
	return m.fn(ctx, next)
}

// NewMiddleware creates a Middleware from a MiddlewareFunc.
func NewMiddleware(fn MiddlewareFunc) Middleware {
	return &MiddlewareFuncAdapter{fn: fn}
}

// Chain executes a chain of middleware with an existing Session followed by a final handler.
// The Session persists across multiple tool calls (server session scope).
func Chain(middlewares []Middleware, sess *session.Session, handler ToolHandler) ToolHandler {
	return func(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
		// Add session to context
		ctx = session.WithSession(ctx, sess)

		mwCtx := &MiddlewareContext{
			Ctx:     ctx,
			Request: req,
			Session: sess,
		}

		// Build the chain from the end
		var chain NextFunc
		chain = func() (*CallToolResult, error) {
			return handler(ctx, req)
		}

		// Wrap each middleware in reverse order
		for i := len(middlewares) - 1; i >= 0; i-- {
			currentMiddleware := middlewares[i]
			next := chain
			chain = func() (*CallToolResult, error) {
				return currentMiddleware.Handle(mwCtx, next)
			}
		}

		// Execute the chain
		return chain()
	}
}
