package mcp

import "net/url"

// ServiceURL returns the streamable-HTTP endpoint of an MCP service. host is the workspace
// URL with its scheme, as config.Config.Host holds it.
func ServiceURL(host, fullName string) string {
	return host + "/ai-gateway/mcp-services/" + url.PathEscape(fullName)
}

// SignInPath is the path of the page where a user signs in to a sign-in-required service, on the workspace host.
const SignInPath = "/mcp-service-login"

// SignInURL returns the page where a user signs in to a sign-in-required service.
func SignInURL(host, fullName, workspaceID string) string {
	query := url.Values{"name": {fullName}, "o": {workspaceID}}
	return host + SignInPath + "?" + query.Encode()
}
