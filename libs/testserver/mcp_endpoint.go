package testserver

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// JSON-RPC codes the fake endpoint answers with.
const (
	rpcCodeParseError     = -32700
	rpcCodeMethodNotFound = -32601
	rpcCodeInvalidParams  = -32602
	// rpcCodeSignInRequired is the gateway's answer to a service that needs a credential the
	// caller does not have.
	rpcCodeSignInRequired = -32006
	// rpcCodeURLElicitation is the gateway's URL elicitation answer, which carries the sign-in page.
	rpcCodeURLElicitation = -32042
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

// McpSignInChallenge is how the fake answers a caller who has no credential for an external service.
type McpSignInChallenge string

const (
	// McpChallengeRPC is JSON-RPC -32006, the gateway's answer before sign-in.
	McpChallengeRPC McpSignInChallenge = "rpc"
	// McpChallengeURL is JSON-RPC -32042 with one URL elicitation to the sign-in page.
	McpChallengeURL McpSignInChallenge = "url"
	// McpChallengeHTTP is HTTP 401 with no JSON-RPC body.
	McpChallengeHTTP McpSignInChallenge = "http"
)

// McpEndpoint fakes POST /ai-gateway/mcp-services/{name}, a streamable HTTP MCP server with one
// tool, echo. A name the workspace has no service for is 404. A service stored with an OAuth source
// connection is external: until a credential exists for it in McpUserCredentials, tools/list and
// tools/call answer the challenge selected by McpSignInChallenge (-32006 by default, as the gateway
// does before sign-in). Notifications and client responses, which have no method or no id, get 202
// with no body.
func (s *FakeWorkspace) McpEndpoint(req Request, name string) Response {
	defer s.LockUnlock()()

	service, ok := s.mcpServiceNamed(name)
	if !ok {
		return Response{StatusCode: http.StatusNotFound}
	}
	var msg mcpRPC
	if err := json.Unmarshal(req.Body, &msg); err != nil {
		return Response{StatusCode: http.StatusBadRequest, Body: rpcError(nil, rpcCodeParseError, "Parse error")}
	}
	if msg.ID == nil || msg.Method == "" {
		return Response{StatusCode: http.StatusAccepted}
	}

	external := mcpServiceIsExternal(service)
	_, signedIn := s.McpUserCredentials[name]
	challenged := external && !signedIn
	switch msg.Method {
	case "initialize":
		return rpcResult(msg.ID, map[string]any{
			"protocolVersion": fakeMcpProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": true}},
			"serverInfo":      map[string]any{"name": "fake_" + name},
			"instructions":    fakeMcpInstructions,
		})
	case "tools/list":
		if challenged {
			return s.signInRequired(msg.ID, name)
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
		if challenged {
			return s.signInRequired(msg.ID, name)
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

// mcpServiceNamed returns the stored service with the given full name. Unity Catalog names are case-insensitive, so the
// match ignores case, as McpServicesList does.
func (s *FakeWorkspace) mcpServiceNamed(name string) (catalog.McpService, bool) {
	for key, service := range s.McpServices {
		if strings.EqualFold(key, name) {
			return service, true
		}
	}
	return catalog.McpService{}, false
}

// mcpServiceIsExternal reports whether the service takes an OAuth source connection.
func mcpServiceIsExternal(ms catalog.McpService) bool {
	return ms.Config != nil && ms.Config.SourceConnection != nil &&
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

// signInRequired answers with the challenge the service is configured for.
func (s *FakeWorkspace) signInRequired(id json.RawMessage, name string) Response {
	switch s.McpSignInChallenge[name] {
	case McpChallengeURL:
		query := url.Values{"name": {name}}
		return Response{Body: map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"error": map[string]any{
				"code":    rpcCodeURLElicitation,
				"message": "This request requires more information.",
				"data": map[string]any{"elicitations": []any{map[string]any{
					"mode":    "url",
					"message": "Sign in to continue.",
					"url":     s.url + "/mcp-service-login?" + query.Encode(),
				}}},
			},
		}}
	case McpChallengeHTTP:
		return Response{StatusCode: http.StatusUnauthorized, Body: "Unauthorized"}
	default:
		return Response{Body: rpcError(id, rpcCodeSignInRequired,
			"The MCP service request was rejected (HTTP 401). Sign-in is required.")}
	}
}
