package api

import "strings"

// workspaceProxyPrefixes lists SDK endpoints that live under accounts/ but
// route to the workspace gateway. Keep this list in sync with workspace-routed
// proxy APIs in the pinned SDK.
var workspaceProxyPrefixes = []string{
	"/api/2.0/accounts/servicePrincipals/",
}

// workspaceProxyExact lists literal SDK endpoints that live under accounts/ but
// route to the workspace gateway.
var workspaceProxyExact = map[string]struct{}{
	"/api/2.0/preview/accounts/access-control/assignable-roles": {},
	"/api/2.0/preview/accounts/access-control/rule-sets":        {},
}

func isWorkspaceProxyPath(path string) bool {
	if _, ok := workspaceProxyExact[path]; ok {
		return true
	}
	for _, prefix := range workspaceProxyPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
