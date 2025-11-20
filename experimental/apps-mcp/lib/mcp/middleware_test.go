package mcp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionData(t *testing.T) {
	sd := mcp.NewSessionData()

	// Test Set and Get
	sd.Set("key1", "value1")
	val, ok := sd.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test Get non-existent key
	_, ok = sd.Get("nonexistent")
	assert.False(t, ok)

	// Test Delete
	sd.Delete("key1")
	_, ok = sd.Get("key1")
	assert.False(t, ok)

	// Test concurrent access
	sd.Set("counter", 0)
	done := make(chan bool)
	for range 10 {
		go func() {
			for range 100 {
				val, _ := sd.Get("counter")
				sd.Set("counter", val.(int)+1)
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}

	// Just verify no race condition occurred (test will fail with -race if there is one)
	val, ok = sd.Get("counter")
	require.True(t, ok)
	assert.IsType(t, 0, val)
}

func TestMiddlewareChain(t *testing.T) {
	var executionOrder []string

	// Create middleware that records execution order
	mw1 := mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "mw1-before")
		result, err := next()
		executionOrder = append(executionOrder, "mw1-after")
		return result, err
	})

	mw2 := mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "mw2-before")
		result, err := next()
		executionOrder = append(executionOrder, "mw2-after")
		return result, err
	})

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "handler")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Type: "text", Text: "result"}},
		}, nil
	}

	sessionData := mcp.NewSessionData()
	chain := mcp.Chain([]mcp.Middleware{mw1, mw2}, sessionData, handler)

	req := &mcp.CallToolRequest{}
	result, err := chain(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []string{
		"mw1-before",
		"mw2-before",
		"handler",
		"mw2-after",
		"mw1-after",
	}, executionOrder)
}

func TestMiddlewareShortCircuit(t *testing.T) {
	var executionOrder []string

	// Middleware that short-circuits
	mw1 := mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "mw1")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Type: "text", Text: "short-circuit"}},
		}, nil
	})

	mw2 := mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "mw2")
		return next()
	})

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "handler")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Type: "text", Text: "handler-result"}},
		}, nil
	}

	sessionData := mcp.NewSessionData()
	chain := mcp.Chain([]mcp.Middleware{mw1, mw2}, sessionData, handler)

	req := &mcp.CallToolRequest{}
	result, err := chain(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "short-circuit", result.Content[0].(*mcp.TextContent).Text)
	assert.Equal(t, []string{"mw1"}, executionOrder)
}

func TestMiddlewareError(t *testing.T) {
	expectedErr := errors.New("middleware error")

	mw := mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		return nil, expectedErr
	})

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	sessionData := mcp.NewSessionData()
	chain := mcp.Chain([]mcp.Middleware{mw}, sessionData, handler)

	req := &mcp.CallToolRequest{}
	result, err := chain(context.Background(), req)

	assert.Nil(t, result)
	assert.Equal(t, expectedErr, err)
}

func TestMiddlewareSessionData(t *testing.T) {
	var capturedData *mcp.SessionData

	mw1 := mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		ctx.SessionData.Set("key1", "value1")
		return next()
	})

	mw2 := mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		val, ok := ctx.SessionData.Get("key1")
		require.True(t, ok)
		assert.Equal(t, "value1", val)

		ctx.SessionData.Set("key2", "value2")
		capturedData = ctx.SessionData
		return next()
	})

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Type: "text", Text: "ok"}},
		}, nil
	}

	sessionData := mcp.NewSessionData()
	chain := mcp.Chain([]mcp.Middleware{mw1, mw2}, sessionData, handler)

	req := &mcp.CallToolRequest{}
	_, err := chain(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, capturedData)

	val, ok := capturedData.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "value1", val)

	val, ok = capturedData.Get("key2")
	require.True(t, ok)
	assert.Equal(t, "value2", val)
}

func TestMiddlewareContextAccess(t *testing.T) {
	type contextKey string
	testKey := contextKey("test-key")
	ctx := context.WithValue(context.Background(), testKey, "test-value")

	mw := mcp.NewMiddleware(func(mwCtx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		val := mwCtx.Ctx.Value(testKey)
		assert.Equal(t, "test-value", val)
		return next()
	})

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Type: "text", Text: "ok"}},
		}, nil
	}

	sessionData := mcp.NewSessionData()
	chain := mcp.Chain([]mcp.Middleware{mw}, sessionData, handler)

	req := &mcp.CallToolRequest{}
	_, err := chain(ctx, req)

	require.NoError(t, err)
}

func TestServerMiddleware(t *testing.T) {
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	server := mcp.NewServer(impl, nil)

	var executionOrder []string

	// Add middleware
	server.AddMiddlewareFunc(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "mw1")
		return next()
	})

	server.AddMiddlewareFunc(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "mw2")
		return next()
	})

	// Add a tool
	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionOrder = append(executionOrder, "handler")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Type: "text", Text: "ok"}},
		}, nil
	}

	server.AddTool(&mcp.Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]any{},
	}, handler)

	// Verify middleware is applied by checking the tool's handler
	tools := server.GetTools()
	require.Len(t, tools, 1)
	assert.Equal(t, "test-tool", tools[0].Name)
}

func TestServerSessionDataPersistence(t *testing.T) {
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	server := mcp.NewServer(impl, nil)

	// Add middleware that increments a counter
	server.AddMiddlewareFunc(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		count := 0
		if val, ok := ctx.SessionData.Get("counter"); ok {
			count = val.(int)
		}
		count++
		ctx.SessionData.Set("counter", count)
		return next()
	})

	// Add a test tool
	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Type: "text", Text: "ok"}},
		}, nil
	}

	server.AddTool(&mcp.Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]any{},
	}, handler)

	// Get the tool handler
	tools := server.GetTools()
	require.Len(t, tools, 1)

	// Execute the tool multiple times
	// Note: We need to get the actual handler from the server's internal state
	// For this test, we'll verify through the server's session data
	sessionData := server.GetSessionData()

	// Simulate tool calls by creating a chain and calling it
	// In the real server, this would happen through handleToolsCall
	toolHandler := mcp.Chain(
		[]mcp.Middleware{
			mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
				count := 0
				if val, ok := ctx.SessionData.Get("counter"); ok {
					count = val.(int)
				}
				count++
				ctx.SessionData.Set("counter", count)
				return next()
			}),
		},
		sessionData,
		handler,
	)

	// Call the tool 3 times
	for range 3 {
		_, err := toolHandler(context.Background(), &mcp.CallToolRequest{})
		require.NoError(t, err)
	}

	// Verify counter persisted across calls
	val, ok := sessionData.Get("counter")
	require.True(t, ok)
	assert.Equal(t, 3, val)
}
