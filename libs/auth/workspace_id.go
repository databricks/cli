package auth

import (
	"context"
	"strconv"

	"github.com/databricks/databricks-sdk-go"
)

// ResolveWorkspaceID returns the workspace ID as a string, preferring the
// value already configured on the client and falling back to a /Me probe.
//
// The fast path short-circuits the API call when w.Config.WorkspaceID is
// set (via --workspace-id, DATABRICKS_WORKSPACE_ID, workspace_id in
// .databrickscfg, ?o=/?w= on a host URL, or any other config source). The
// CLI-only "none" sentinel is treated as unset so it never leaks as a
// routing identifier.
//
// The fallback path delegates to w.CurrentWorkspaceID, which reads the
// X-Databricks-Org-Id response header on /api/2.0/preview/scim/v2/Me and
// parses it as int64. The numeric constraint is enforced by the SDK on
// that path; the helper just stringifies the result.
//
// Compared to calling w.CurrentWorkspaceID directly, the string return
// type lets callers pass the value to URL builders, env vars, and other
// string-typed sinks without a manual strconv.FormatInt step.
func ResolveWorkspaceID(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	if id := w.Config.WorkspaceID; id != "" && id != WorkspaceIDNone {
		return id, nil
	}
	id, err := w.CurrentWorkspaceID(ctx)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}
