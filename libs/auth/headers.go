package auth

import (
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
)

// WorkspaceIDHeader is the request header name used to route workspace-scoped
// API calls to the correct workspace on unified ("SPOG") hosts. The platform
// gateway also accepts the legacy X-Databricks-Org-Id header for rollback
// safety. Generated SDK service methods set this header per-call when
// cfg.WorkspaceID is populated; CLI code paths that call client.Do directly
// need to set it themselves.
const WorkspaceIDHeader = "X-Databricks-Workspace-Id"

// WorkspaceIDHeaders returns a map suitable as the headers argument to
// client.DatabricksClient.Do, populated with the workspace routing header
// when cfg.WorkspaceID is set. Returns nil when the workspace ID is unset
// or holds the CLI-only "none" sentinel, so callers can pass the result
// through without conditional checks.
func WorkspaceIDHeaders(cfg *sdkconfig.Config) map[string]string {
	if cfg == nil {
		return nil
	}
	wsID := cfg.WorkspaceID
	if wsID == "" || wsID == WorkspaceIDNone {
		return nil
	}
	return map[string]string{
		WorkspaceIDHeader: wsID,
	}
}
