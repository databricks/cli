package grants

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "grants",
	Short: `In Unity Catalog, data is secure by default.`,
	Long: `In Unity Catalog, data is secure by default. Initially, users have no access
  to data in a metastore. Access can be granted by either a metastore admin, the
  owner of an object, or the owner of the catalog or schema that contains the
  object. Securable objects in Unity Catalog are hierarchical and privileges are
  inherited downward.
  
  Initially, users have no access to data in a metastore. Access can be granted
  by either a metastore admin, the owner of an object, or the owner of the
  catalog or schema that contains the object.
  
  Securable objects in Unity Catalog are hierarchical and privileges are
  inherited downward. This means that granting a privilege on the catalog
  automatically grants the privilege to all current and future objects within
  the catalog. Similarly, privileges granted on a schema are inherited by all
  current and future objects within that schema.`,
}

var getReq unitycatalog.GetGrantRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.FullName, "full-name", "", `Required.`)
	getCmd.Flags().StringVar(&getReq.Principal, "principal", "", `Optional.`)
	getCmd.Flags().StringVar(&getReq.SecurableType, "securable-type", "", `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get permissions.`,
	Long: `Get permissions.
  
  Gets the permissions for a Securable type.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Grants.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var updateReq unitycatalog.UpdatePermissions

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	// TODO: array: changes
	updateCmd.Flags().StringVar(&updateReq.FullName, "full-name", "", `Required.`)
	updateCmd.Flags().StringVar(&updateReq.Principal, "principal", "", `Optional.`)
	updateCmd.Flags().StringVar(&updateReq.SecurableType, "securable-type", "", `Required.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update permissions.`,
	Long: `Update permissions.
  
  Updates the permissions for a Securable type.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Grants.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}

// end service Grants
