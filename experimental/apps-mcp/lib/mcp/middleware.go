package mcp

import (
	"context"
	"sync"
)

// SessionData provides thread-safe storage for session-scoped data that middleware can read/write.
type SessionData struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewSessionData creates a new SessionData instance.
func NewSessionData() *SessionData {
	return &SessionData{
		data: make(map[string]any),
	}
}

// Get retrieves a value from session data.
func (sd *SessionData) Get(key string) (any, bool) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	valRaw, ok := sd.data[key]
	if !ok {
		return nil, ok
	}
	return valRaw, true
}

// GetBool retrieves a value from session data and casts it to a boolean.
func (sd *SessionData) GetBool(key string) (bool, bool) {
	val, ok := sd.Get(key)
	if !ok {
		return false, ok
	}
	return val.(bool), true
}

// Set stores a value in session data.
func (sd *SessionData) Set(key string, value any) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.data[key] = value
}

// Delete removes a value from session data.
func (sd *SessionData) Delete(key string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	delete(sd.data, key)
}

// MiddlewareContext provides context for middleware execution.
type MiddlewareContext struct {
	Ctx         context.Context
	Request     *CallToolRequest
	SessionData *SessionData
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

// Chain executes a chain of middleware with an existing SessionData followed by a final handler.
// The SessionData persists across multiple tool calls (server session scope).
func Chain(middlewares []Middleware, sessionData *SessionData, handler ToolHandler) ToolHandler {
	return func(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
		mwCtx := &MiddlewareContext{
			Ctx:         ctx,
			Request:     req,
			SessionData: sessionData,
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
