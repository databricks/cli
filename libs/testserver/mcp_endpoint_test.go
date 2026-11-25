package testserver

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
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
	mcpServicesTestSeed(workspace, name, "Genie", "")

	reply := mcpEndpointTestCall(t, workspace, name, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Nil(t, reply["error"])
	result := reply["result"].(map[string]any)
	assert.Equal(t, "echo", result["tools"].([]any)[0].(map[string]any)["name"])

	reply = mcpEndpointTestCall(t, workspace, name,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}`)
	content := reply["result"].(map[string]any)["content"].([]any)
	assert.Equal(t, "hi", content[0].(map[string]any)["text"])
}

// TestMcpEndpointMatchesServiceNameWithoutCase checks that the endpoint matches a service name whatever its case, as
// Unity Catalog names are case-insensitive and the listing already matches them that way.
func TestMcpEndpointMatchesServiceNameWithoutCase(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	mcpServicesTestSeed(workspace, "system.ai.genie_one_mcp", "Genie", "")

	reply := mcpEndpointTestCall(t, workspace, "SYSTEM.AI.Genie_One_MCP", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Nil(t, reply["error"])
	assert.NotNil(t, reply["result"])
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

// TestMcpEndpointUnknownServiceIsNotFound checks that a name no service has is 404, whatever the message. The name is
// one letter short of the seeded service, so it also shows that the endpoint matches whole names, not prefixes.
func TestMcpEndpointUnknownServiceIsNotFound(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	mcpServicesTestSeed(workspace, "system.ai.genie_one_mcp", "Genie", "")

	for _, body := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
	} {
		response := workspace.McpEndpoint(Request{Body: []byte(body)}, "system.ai.genie_one_mc")
		assert.Equal(t, 404, response.StatusCode, body)
		assert.Nil(t, response.Body)
	}
}

// TestMcpEndpointNoBodyForNotificationsAndResponses checks that a notification, and a client's answer to a request from
// the server (an id and no method), get 202 with no body.
func TestMcpEndpointNoBodyForNotificationsAndResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "notification", body: `{"jsonrpc":"2.0","method":"notifications/initialized"}`},
		{name: "client response", body: `{"jsonrpc":"2.0","id":7,"result":{}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workspace := NewFakeWorkspace("http://test", "dbapi123")
			mcpServicesTestSeed(workspace, "system.ai.genie_one_mcp", "Genie", "")
			response := workspace.McpEndpoint(Request{Body: []byte(tc.body)}, "system.ai.genie_one_mcp")
			assert.Equal(t, 202, response.StatusCode)
			assert.Nil(t, response.Body)
		})
	}
}

func TestMcpEndpointPassesThroughOnceCredentialExists(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	const name = "system.ai.github"
	mcpServicesTestSeed(workspace, name, "GitHub", mcpServicesTestGitHubProvider)
	const list = `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	assert.EqualValues(t, -32006, mcpEndpointTestCall(t, workspace, name, list)["error"].(map[string]any)["code"])

	workspace.McpUserCredentialCreate(name)

	reply := mcpEndpointTestCall(t, workspace, name, list)
	assert.Nil(t, reply["error"])
	assert.NotNil(t, reply["result"])

	workspace.McpUserCredentials = map[string]catalog.McpServiceUserMappedCredential{}
	assert.EqualValues(t, -32006, mcpEndpointTestCall(t, workspace, name, list)["error"].(map[string]any)["code"])
}

func TestMcpEndpointCredentialMatchesNameWithoutCase(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	const name = "system.ai.github"
	mcpServicesTestSeed(workspace, name, "GitHub", mcpServicesTestGitHubProvider)
	workspace.McpUserCredentialCreate(name)
	const list = `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	reply := mcpEndpointTestCall(t, workspace, "SYSTEM.AI.GITHUB", list)
	assert.Nil(t, reply["error"], "the service and the credential match the name without case")
}
