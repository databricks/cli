package testserver

import (
	"encoding/json"
	"net/http"
)

// JSON-RPC codes the fake endpoint answers with.
const (
	rpcCodeParseError     = -32700
	rpcCodeMethodNotFound = -32601
	rpcCodeInvalidParams  = -32602
	// rpcCodeSignInRequired is the gateway's answer to a service that needs a credential the
	// caller does not have.
	rpcCodeSignInRequired = -32006
)

const (
	jsonRPCVersion         = "2.0"
	fakeMcpProtocolVersion = "2025-06-18"
	fakeMcpInstructions    = "Fake Databricks MCP service for tests."
	fakeMcpEchoTool        = "echo"
)

// mcpRPC is the part of a client message the fake reads: the id, the method, and the tool call's name and arguments.
type mcpRPC struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
	Params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"params"`
}

// McpEndpoint fakes POST /ai-gateway/mcp-services/{name}, a streamable HTTP MCP server with one
// tool, echo. A service stored with an OAuth source connection is external: no credential is
// ever mapped to it here, so tools/list and tools/call answer -32006 as the gateway does before
// sign-in. Notifications get 202 with no body.
func (s *FakeWorkspace) McpEndpoint(req Request, name string) Response {
	defer s.LockUnlock()()

	var msg mcpRPC
	if err := json.Unmarshal(req.Body, &msg); err != nil {
		return Response{StatusCode: http.StatusBadRequest, Body: rpcError(nil, rpcCodeParseError, "Parse error")}
	}
	if msg.ID == nil {
		return Response{StatusCode: http.StatusAccepted}
	}

	external := s.mcpServiceIsExternal(name)
	switch msg.Method {
	case "initialize":
		return rpcResult(msg.ID, map[string]any{
			"protocolVersion": fakeMcpProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": true}},
			"serverInfo":      map[string]any{"name": "fake_" + name},
			"instructions":    fakeMcpInstructions,
		})
	case "tools/list":
		if external {
			return signInRequired(msg.ID)
		}
		return rpcResult(msg.ID, map[string]any{"tools": []any{
			map[string]any{
				"name":        fakeMcpEchoTool,
				"description": "Returns the text argument",
				"inputSchema": map[string]any{
					"type":       "object",
					"properties": map[string]any{"text": map[string]any{"type": "string"}},
				},
			},
		}})
	case "tools/call":
		if external {
			return signInRequired(msg.ID)
		}
		if msg.Params.Name != fakeMcpEchoTool {
			return Response{Body: rpcError(msg.ID, rpcCodeInvalidParams, "Unknown tool: "+msg.Params.Name)}
		}
		text, _ := msg.Params.Arguments["text"].(string)
		return rpcResult(msg.ID, map[string]any{
			"content": []any{map[string]any{"type": "text", "text": text}},
			"isError": false,
		})
	default:
		return Response{Body: rpcError(msg.ID, rpcCodeMethodNotFound, "Method not found: "+msg.Method)}
	}
}

// mcpServiceIsExternal reports whether the stored service name takes an OAuth source connection.
func (s *FakeWorkspace) mcpServiceIsExternal(name string) bool {
	ms, ok := s.McpServices[name]
	return ok && ms.Config != nil && ms.Config.SourceConnection != nil &&
		ms.Config.SourceConnection.Options["oauth_provider"] != ""
}

// rpcResult is the response to the request with id, carrying result.
func rpcResult(id json.RawMessage, result any) Response {
	return Response{Body: map[string]any{"jsonrpc": jsonRPCVersion, "id": id, "result": result}}
}

// rpcError is the error response to the request with id.
func rpcError(id json.RawMessage, code int, message string) map[string]any {
	return map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"error":   map[string]any{"code": code, "message": message, "data": nil},
	}
}

func signInRequired(id json.RawMessage) Response {
	return Response{Body: rpcError(id, rpcCodeSignInRequired,
		"The MCP service request was rejected (HTTP 401). Sign-in is required.")}
}
