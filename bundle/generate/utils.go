package generate

import (
	"path"
	"strings"
)

// ensureWorkspacePrefix re-adds the /Workspace prefix that the Genie GET API
// strips from parent_path, so the generated config matches the convention used
// in hand-written bundles and in deployment state (mirrors the equivalent
// helper in bundle/direct/dresources/dashboard.go).
func ensureWorkspacePrefix(parentPath string) string {
	if parentPath == "/Workspace" || strings.HasPrefix(parentPath, "/Workspace/") {
		return parentPath
	}
	return path.Join("/Workspace", parentPath)
}
