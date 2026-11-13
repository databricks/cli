package testserver

import (
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mcpServicesTestGitHubProvider is the oauth_provider the API reports for GitHub.
const mcpServicesTestGitHubProvider = "OAUTH_PROVIDER_GITHUB_MCP"

// mcpServicesTestList calls the list handler with the given query string.
func mcpServicesTestList(t *testing.T, workspace *FakeWorkspace, rawQuery string) []catalog.McpService {
	t.Helper()
	u, err := url.Parse("/api/2.1/unity-catalog/mcp-services?" + rawQuery)
	require.NoError(t, err)

	response := workspace.McpServicesList(Request{URL: u})
	require.Equal(t, 0, response.StatusCode)
	body, ok := response.Body.(catalog.ListMcpServicesResponse)
	require.True(t, ok)
	return body.McpServices
}

// mcpServicesTestWorkspace returns a fake workspace with two system.ai services and one customer service.
func mcpServicesTestWorkspace(t *testing.T) *FakeWorkspace {
	t.Helper()
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	mcpServicesTestSeed(workspace, "system.ai.genie_one_mcp", "Genie", "")
	mcpServicesTestSeed(workspace, "system.ai.github", "GitHub", mcpServicesTestGitHubProvider)
	mcpServicesTestSeed(workspace, "main.tools.crm", "CRM", mcpServicesTestGitHubProvider)
	return workspace
}

func TestMcpServicesListFiltersByParent(t *testing.T) {
	workspace := mcpServicesTestWorkspace(t)

	services := mcpServicesTestList(t, workspace, "parent=schemas/system.ai&view=FULL")
	require.Len(t, services, 2)
	assert.Equal(t, "mcp-services/system.ai.genie_one_mcp", services[0].Name)
	assert.Equal(t, "mcp-services/system.ai.github", services[1].Name)
}

func TestMcpServicesListMatchesParentInAnyCase(t *testing.T) {
	workspace := mcpServicesTestWorkspace(t)

	services := mcpServicesTestList(t, workspace, "parent=schemas/System.AI&view=FULL")
	require.Len(t, services, 2)
	assert.Equal(t, "mcp-services/system.ai.genie_one_mcp", services[0].Name)
}

func TestMcpServicesListViewFullKeepsSourceConnection(t *testing.T) {
	workspace := mcpServicesTestWorkspace(t)

	services := mcpServicesTestList(t, workspace, "parent=schemas/system.ai&view=FULL")
	require.Len(t, services, 2)
	assert.Nil(t, services[0].Config)
	require.NotNil(t, services[1].Config)
	require.NotNil(t, services[1].Config.SourceConnection)
	assert.Equal(t, mcpServicesTestGitHubProvider, services[1].Config.SourceConnection.Options["oauth_provider"])
}

func TestMcpServicesListViewBasicOmitsSourceConnection(t *testing.T) {
	workspace := mcpServicesTestWorkspace(t)

	for _, query := range []string{"parent=schemas/system.ai&view=BASIC", "parent=schemas/system.ai"} {
		services := mcpServicesTestList(t, workspace, query)
		require.Len(t, services, 2, query)
		require.NotNil(t, services[1].Config, query)
		assert.Nil(t, services[1].Config.SourceConnection, query)
	}

	// BASIC must not strip the stored service, or a later FULL read would lose sign-in details.
	services := mcpServicesTestList(t, workspace, "parent=schemas/system.ai&view=FULL")
	require.NotNil(t, services[1].Config)
	require.NotNil(t, services[1].Config.SourceConnection)
}

// mcpServicesTestSeed stores an MCP service under fullName (catalog.schema.leaf).
// An empty oauthProvider seeds an internal Databricks service with no source connection;
// otherwise the service is backed by a source connection with that OAuth provider.
func mcpServicesTestSeed(s *FakeWorkspace, fullName, comment, oauthProvider string) {
	ms := catalog.McpService{
		Name:    mcpServicesNamePrefix + fullName,
		Comment: comment,
	}
	if oauthProvider != "" {
		ms.Config = &catalog.McpServiceConfig{
			SourceConnection: &catalog.McpServiceConfigSourceConnection{
				Name:    "connections/" + fullName,
				Options: map[string]string{"oauth_provider": oauthProvider},
			},
		}
	}
	s.McpServices[fullName] = ms
}
