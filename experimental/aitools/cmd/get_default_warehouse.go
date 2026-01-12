package mcp

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/middlewares"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

type warehouseInfo struct {
	Id    string    `json:"id"`
	Name  string    `json:"name"`
	State sql.State `json:"state"`
}

func newGetDefaultWarehouseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-default-warehouse",
		Short: "Get the default warehouse ID",
		Long: `Get the default warehouse ID for the current workspace.

The command auto-detects an available warehouse unless DATABRICKS_WAREHOUSE_ID is set.

Returns warehouse ID of the default warehouse. Use --output json to get the full warehouse info including name and state.`,
		Example: `  # Get warehouse ID in text format (default)
  databricks experimental aitools tools get-default-warehouse
  # Output: abc123def456...

  # Get full warehouse info including name and state in JSON format
  databricks experimental aitools tools get-default-warehouse --output json
  # Output: {"id":"abc123def456...","name":"My Warehouse","state":"RUNNING"}`,
		Args:    cobra.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			// set up session with client for middleware compatibility
			sess := session.NewSession()
			sess.Set(middlewares.DatabricksClientKey, w)
			ctx = session.WithSession(ctx, sess)

			warehouse, err := middlewares.GetWarehouseEndpoint(ctx)
			if err != nil {
				return err
			}

			info := warehouseInfo{
				Id:    warehouse.Id,
				Name:  warehouse.Name,
				State: warehouse.State,
			}

			return cmdio.RenderWithTemplate(ctx, info, "", "{{.Id}}\n")
		},
	}

	return cmd
}
