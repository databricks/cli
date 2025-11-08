package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/databricks/cli/cmd/mcp/tools"
	"github.com/databricks/cli/libs/log"
)

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

// MCPServer implements the Model Context Protocol server.
type MCPServer struct {
	ctx         context.Context
	in          io.Reader
	out         io.Writer
	cachedRoots []WorkspaceRoot
}

// NewMCPServer creates a new MCP server instance.
func NewMCPServer(ctx context.Context) *MCPServer {
	return &MCPServer{
		ctx: ctx,
		in:  os.Stdin,
		out: os.Stdout,
	}
}

// Start starts the MCP server and processes requests.
func (s *MCPServer) Start() error {
	// Note: No logging in server mode - it interferes with JSON-RPC over stdout/stdin

	scanner := bufio.NewScanner(s.in)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error", nil)
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
	// Note: No logging - interferes with JSON-RPC

	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	default:
		s.sendError(req.ID, -32601, "Method not found", nil)
	}
}

// handleInitialize handles the initialize request.
func (s *MCPServer) handleInitialize(req *JSONRPCRequest) {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "databricks-cli",
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
	tools := []map[string]any{
		{
			"name":        "init_project",
			"description": "Initialize a new Databricks project (an app, dashboard, job, pipeline, ETL application, etc.). Use to create a new project and to get information about how to adjust it.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "A name for this project in snake_case. Ask the user about this if it's not clear from the context.",
					},
					"project_path": map[string]any{
						"type":        "string",
						"description": "A fully qualified path for the directory to create the new project in. If the current directory is fully empty it may make sense to just put it in the current directory; otherwise it could be a subdirectory.",
					},
				},
				"required": []string{"project_name", "project_path"},
			},
		},
		{
			"name":        "analyze_project",
			"description": "Determine what the current project is about and what actions can be performed on it. Mandatory: use this for more guidance whenever the user asks things like 'run or deploy ...' or 'add ..., like add a pipeline or a job or an app' or 'change the app/dashboard/pipeline job to ...' or 'open ... in my browser' or 'preview ...'.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_path": map[string]any{
						"type":        "string",
						"description": "A fully qualified path of the project to operate on. By default, the current directory.",
					},
				},
				"required": []string{"project_path"},
			},
		},
	}

	result := map[string]any{
		"tools": tools,
	}

	s.sendResponse(req.ID, result)
}

// handleToolsCall handles the tools/call request.
func (s *MCPServer) handleToolsCall(req *JSONRPCRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	switch params.Name {
	case "init_project":
		s.handleInitProject(req.ID, params.Arguments)
	case "analyze_project":
		s.handleAnalyzeProject(req.ID, params.Arguments)
	default:
		s.sendError(req.ID, -32602, "Unknown tool", params.Name)
	}
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

// handleInitProject implements the init_project tool.
func (s *MCPServer) handleInitProject(id any, args map[string]any) {
	projectName, ok := args["project_name"].(string)
	if !ok {
		s.sendError(id, -32602, "Missing or invalid project_name parameter", nil)
		return
	}

	projectPath, ok := args["project_path"].(string)
	if !ok {
		s.sendError(id, -32602, "Missing or invalid project_path parameter", nil)
		return
	}

	result, err := tools.InitProject(s.ctx, tools.InitProjectArgs{
		ProjectName: projectName,
		ProjectPath: projectPath,
	})
	if err != nil {
		s.sendError(id, -32603, "Failed to initialize project", err.Error())
		return
	}

	s.sendResponse(id, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": result,
			},
		},
	})
}

// handleAnalyzeProject implements the analyze_project tool.
func (s *MCPServer) handleAnalyzeProject(id any, args map[string]any) {
	projectPath, ok := args["project_path"].(string)
	if !ok {
		s.sendError(id, -32602, "Missing or invalid project_path parameter", nil)
		return
	}

	result, err := tools.AnalyzeProject(s.ctx, tools.AnalyzeProjectArgs{
		ProjectPath: projectPath,
	})
	if err != nil {
		s.sendError(id, -32603, "Failed to analyze project", err.Error())
		return
	}

	s.sendResponse(id, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": result,
			},
		},
	})
}

func runServer(ctx context.Context) error {
	server := NewMCPServer(ctx)
	return server.Start()
}
