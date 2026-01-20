package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/databricks/cli/experimental/aitools/lib/errors"
	"github.com/databricks/cli/experimental/aitools/lib/session"
)

// Server is an MCP server that manages tools and handles requests.
type Server struct {
	impl        *Implementation
	tools       map[string]*serverTool
	toolsMu     sync.RWMutex
	transport   *StdioTransport
	initialized bool
	middlewares []Middleware
	mwMu        sync.RWMutex
	session     *session.Session
}

// serverTool represents a registered tool with its handler.
type serverTool struct {
	tool    *Tool
	handler ToolHandler
}

// NewServer creates a new MCP server.
// If sess is nil, a new session will be created.
func NewServer(impl *Implementation, options any, sess *session.Session) *Server {
	if sess == nil {
		sess = session.NewSession()
	}
	return &Server{
		impl:    impl,
		tools:   make(map[string]*serverTool),
		session: sess,
	}
}

// AddMiddleware registers middleware to be applied to all tool calls.
// Middleware is executed in the order it is registered.
func (s *Server) AddMiddleware(mw Middleware) {
	s.mwMu.Lock()
	defer s.mwMu.Unlock()
	s.middlewares = append(s.middlewares, mw)
}

// AddMiddlewareFunc is a convenience method to register a middleware function.
func (s *Server) AddMiddlewareFunc(fn MiddlewareFunc) {
	s.AddMiddleware(NewMiddleware(fn))
}

// GetSession returns the server's Session.
// This persists across all tool calls during the server's lifetime.
func (s *Server) GetSession() *session.Session {
	return s.session
}

// AddTool registers a tool with a low-level handler.
// This is the internal method used by the typed AddTool function.
func (s *Server) AddTool(tool *Tool, handler ToolHandler) {
	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()

	// Wrap the handler with middleware chain using server's session
	s.mwMu.RLock()
	wrappedHandler := Chain(s.middlewares, s.session, handler)
	s.mwMu.RUnlock()

	s.tools[tool.Name] = &serverTool{
		tool:    tool,
		handler: wrappedHandler,
	}
}

// GetTools returns all registered tools.
func (s *Server) GetTools() []*Tool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()

	tools := make([]*Tool, 0, len(s.tools))
	for _, st := range s.tools {
		tools = append(tools, st.tool)
	}
	return tools
}

// Run starts the MCP server with the given transport.
func (s *Server) Run(ctx context.Context, transport *StdioTransport) error {
	s.transport = transport

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := transport.Read(ctx)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		resp := s.handleRequest(ctx, req)
		if resp != nil {
			if err := transport.Write(ctx, resp); err != nil {
				return err
			}
		}
	}
}

// handleRequest processes a JSON-RPC request and returns a response.
func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodToolsList:
		return s.handleToolsList(req)
	case MethodToolsCall:
		return s.handleToolsCall(ctx, req)
	case MethodPing:
		return s.handlePing(req)
	case MethodNotification:
		// Notifications don't require a response
		s.initialized = true
		return nil
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    errors.CodeMethodNotFound,
				Message: "method not found: " + req.Method,
			},
		}
	}
}

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
		},
		ServerInfo: *s.impl,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    errors.CodeInternalError,
				Message: fmt.Sprintf("failed to marshal result: %v", err),
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  data,
	}
}

// handleToolsList handles the tools/list request.
func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, st := range s.tools {
		tools = append(tools, *st.tool)
	}

	result := ListToolsResult{
		Tools: tools,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return CreateNewErrorResponse(req.ID, errors.CodeInternalError, fmt.Sprintf("failed to marshal result: %v", err))
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  data,
	}
}

// handleToolsCall handles the tools/call request.
func (s *Server) handleToolsCall(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return CreateNewErrorResponse(req.ID, errors.CodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
	}

	s.toolsMu.RLock()
	st, ok := s.tools[params.Name]
	s.toolsMu.RUnlock()

	if !ok {
		return CreateNewErrorResponse(req.ID, errors.CodeInvalidParams, "tool not found: "+params.Name)
	}

	toolReq := &CallToolRequest{
		ID:     req.ID,
		Tool:   st.tool,
		Params: params,
	}

	result, err := st.handler(ctx, toolReq)
	if err != nil {
		result = CreateNewTextContentResultError(err)
	}

	// Convert Content slice to []any for JSON marshaling
	content := make([]any, len(result.Content))
	for i, c := range result.Content {
		content[i] = c
	}

	resultData := struct {
		Content []any `json:"content"`
		IsError bool  `json:"isError,omitempty"`
	}{
		Content: content,
		IsError: result.IsError,
	}

	data, err := json.Marshal(resultData)
	if err != nil {
		return CreateNewErrorResponse(req.ID, errors.CodeInternalError, fmt.Sprintf("failed to marshal result: %v", err))
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  data,
	}
}

// handlePing handles the ping request.
func (s *Server) handlePing(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  json.RawMessage("{}"),
	}
}
