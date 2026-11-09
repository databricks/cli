package template

import (
	"context"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
)

// HelpersProfileEnvVar names a profile for the requests that read the workspace while a template
// renders: the helpers such as workspace_host and user_name, and the warehouse_path prompt. The
// selected profile still provides .workspace and writes the generated files.
//
// DABs in the Workspace selects the profile for the host of the page, which can be a host shared
// by several workspaces, and sets this to a profile for the workspace's own host, so that
// templates that use workspace_host generate that host.
const HelpersProfileEnvVar = "DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE"

type helpersClientKey struct{}

// WithHelpersClient returns a context in which the reads of HelpersProfileEnvVar use w instead of
// the selected client.
func WithHelpersClient(ctx context.Context, w *databricks.WorkspaceClient) context.Context {
	return context.WithValue(ctx, helpersClientKey{}, w)
}

// helpersClient returns the client set by WithHelpersClient, or the selected client.
func helpersClient(ctx context.Context) *databricks.WorkspaceClient {
	if w, ok := ctx.Value(helpersClientKey{}).(*databricks.WorkspaceClient); ok {
		return w
	}
	return cmdctx.WorkspaceClient(ctx)
}
