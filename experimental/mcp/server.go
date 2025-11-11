package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/databricks/cli/experimental/mcp/tools"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the MCP server (used by coding agents)",
		Long:  `Start the Databricks CLI MCP server. This command is typically invoked by coding agents.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd.Context())
		},
	}

	return cmd
}

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON-RPC 2.0 error codes.
const (
	jsonRPCParseError     = -32700
	jsonRPCInvalidRequest = -32600
	jsonRPCMethodNotFound = -32601
	jsonRPCInvalidParams  = -32602
	jsonRPCInternalError  = -32603
)

// MCPServer implements the Model Context Protocol server.
type MCPServer struct {
	ctx        context.Context
	in         io.Reader
	out        io.Writer
	toolsMap   map[string]tools.ToolHandler
	clientName string
}

// getAllTools returns all tools (definitions + handlers) for the MCP server.
func getAllTools() []tools.Tool {
	return []tools.Tool{
		tools.InvokeDatabricksCLITool,
		tools.InitProjectTool,
		tools.AnalyzeProjectTool,
		tools.AddProjectResourceTool,
		tools.ExploreTool,
	}
}

// NewMCPServer creates a new MCP server instance.
func NewMCPServer(ctx context.Context) *MCPServer {
	allTools := getAllTools()
	toolsMap := make(map[string]tools.ToolHandler, len(allTools))
	for _, tool := range allTools {
		toolsMap[tool.Definition.Name] = tool.Handler
	}

	return &MCPServer{
		ctx:      ctx,
		in:       os.Stdin,
		out:      os.Stdout,
		toolsMap: toolsMap,
	}
}

// Start starts the MCP server and processes requests.
// Note: No logging in server mode as it interferes with JSON-RPC over stdout/stdin.
func (s *MCPServer) Start() error {
	scanner := bufio.NewScanner(s.in)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, jsonRPCParseError, "Parse error", nil)
			continue
		}

		s.handleRequest(&req)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	return nil
}

// handleRequest processes an incoming JSON-RPC request.
func (s *MCPServer) handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	default:
		s.sendError(req.ID, jsonRPCMethodNotFound, "Method not found", nil)
	}
}

// handleInitialize handles the initialize request.
func (s *MCPServer) handleInitialize(req *JSONRPCRequest) {
	// Parse clientInfo from the request
	var params struct {
		ClientInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}
	if req.Params != nil {
		_ = json.Unmarshal(req.Params, &params)
		s.clientName = params.ClientInfo.Name
	}

	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "databricks-mcp",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{
			"tools": map[string]bool{},
		},
	}

	s.sendResponse(req.ID, result)
}

// handleToolsList handles the tools/list request.
func (s *MCPServer) handleToolsList(req *JSONRPCRequest) {
	allTools := getAllTools()
	mcpTools := make([]map[string]any, len(allTools))
	for i, tool := range allTools {
		mcpTools[i] = map[string]any{
			"name":        tool.Definition.Name,
			"description": tool.Definition.Description,
			"inputSchema": tool.Definition.InputSchema,
		}
	}

	s.sendResponse(req.ID, map[string]any{"tools": mcpTools})
}

// handleToolsCall handles the tools/call request.
func (s *MCPServer) handleToolsCall(req *JSONRPCRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, jsonRPCInvalidParams, "Invalid params", err.Error())
		return
	}

	handler, ok := s.toolsMap[params.Name]
	if !ok {
		s.sendError(req.ID, jsonRPCInvalidParams, "Unknown tool", params.Name)
		return
	}

	// Add client name to context
	ctx := tools.SetClientName(s.ctx, s.clientName)

	result, err := handler(ctx, params.Arguments)
	if err != nil {
		s.sendError(req.ID, jsonRPCInternalError, "Tool execution failed: "+err.Error(), nil)
		return
	}

	s.sendResponse(req.ID, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": result,
			},
		},
	})
}

// sendResponse sends a JSON-RPC response.
func (s *MCPServer) sendResponse(id, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Errorf(s.ctx, "Failed to marshal response: %v", err)
		return
	}

	_, _ = s.out.Write(append(data, '\n'))
}

// sendError sends a JSON-RPC error response.
func (s *MCPServer) sendError(id any, code int, message string, data any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		log.Errorf(s.ctx, "Failed to marshal error response: %v", err)
		return
	}

	_, _ = s.out.Write(append(respData, '\n'))
}

func runServer(ctx context.Context) error {
	server := NewMCPServer(ctx)
	return server.Start()
}
