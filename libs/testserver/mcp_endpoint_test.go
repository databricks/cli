package testserver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mcpEndpointTestCall posts one JSON-RPC message to the fake endpoint for name and decodes the reply.
func mcpEndpointTestCall(t *testing.T, workspace *FakeWorkspace, name, message string) map[string]any {
	t.Helper()
	response := workspace.McpEndpoint(Request{Body: []byte(message)}, name)
	require.Equal(t, 0, response.StatusCode)
	encoded, err := json.Marshal(response.Body)
	require.NoError(t, err)
	var reply map[string]any
	require.NoError(t, json.Unmarshal(encoded, &reply))
	return reply
}

func TestMcpEndpointAnswersInternalService(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	const name = "system.ai.genie_one_mcp"

	reply := mcpEndpointTestCall(t, workspace, name, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Nil(t, reply["error"])
	result := reply["result"].(map[string]any)
	assert.Equal(t, "echo", result["tools"].([]any)[0].(map[string]any)["name"])

	reply = mcpEndpointTestCall(t, workspace, name,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}`)
	content := reply["result"].(map[string]any)["content"].([]any)
	assert.Equal(t, "hi", content[0].(map[string]any)["text"])
}

func TestMcpEndpointRejectsExternalServiceWithoutCredential(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	const name = "system.ai.github"
	mcpServicesTestSeed(workspace, name, "GitHub", mcpServicesTestGitHubProvider)

	reply := mcpEndpointTestCall(t, workspace, name, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	errBody := reply["error"].(map[string]any)
	assert.EqualValues(t, -32006, errBody["code"])
	assert.Nil(t, reply["result"])
}

func TestMcpEndpointNotificationGetsNoBody(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	response := workspace.McpEndpoint(Request{Body: []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)}, "system.ai.genie_one_mcp")
	assert.Equal(t, 202, response.StatusCode)
	assert.Nil(t, response.Body)
}

// A client's answer to a request from the server has an id and no method. It is not a request, so it gets 202 with no body.
func TestMcpEndpointClientResponseGetsNoBody(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	response := workspace.McpEndpoint(Request{Body: []byte(`{"jsonrpc":"2.0","id":7,"result":{}}`)}, "system.ai.genie_one_mcp")
	assert.Equal(t, 202, response.StatusCode)
	assert.Nil(t, response.Body)
}
