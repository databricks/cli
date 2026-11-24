package mcp

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/databrickscfg/profile"
)

// ServiceURL returns the streamable-HTTP endpoint of an MCP service. host is the workspace
// URL with its scheme, as config.Config.Host holds it.
func ServiceURL(host, fullName string) string {
	return host + mcpconfig.MCPServicesPathPrefix + url.PathEscape(fullName)
}

// SignInPath is the path of the page where a user signs in to a sign-in-required service, on the workspace host.
const SignInPath = "/mcp-service-login"

// SignInURL returns the page where a user signs in to a sign-in-required service. The page is
// chosen by workspace ID, so an empty ID or profile.WorkspaceIDNone is an error. A numeric ID is
// passed as o=, which accepts only numbers; any other ID is passed as w=, which accepts any value.
func SignInURL(host, fullName, workspaceID string) (string, error) {
	if workspaceID == "" || workspaceID == profile.WorkspaceIDNone {
		return "", fmt.Errorf("cannot build the sign-in URL for %s: the workspace ID is not known", fullName)
	}
	key := "o"
	if _, err := strconv.ParseInt(workspaceID, 10, 64); err != nil {
		key = "w"
	}
	query := url.Values{"name": {fullName}, key: {workspaceID}}
	return host + SignInPath + "?" + query.Encode(), nil
}
