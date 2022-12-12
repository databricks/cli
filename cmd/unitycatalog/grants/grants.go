// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package grants

import (
	"github.com/databricks/bricks/lib/jsonflag"
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

// start get command

var getReq unitycatalog.GetGrantRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Principal, "principal", getReq.Principal, `Optional.`)

}

var getCmd = &cobra.Command{
	Use:   "get SECURABLE_TYPE FULL_NAME",
	Short: `Get permissions.`,
	Long: `Get permissions.
  
  Gets the permissions for a Securable type.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getReq.SecurableType = args[0]
		getReq.FullName = args[1]

		response, err := w.Grants.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq unitycatalog.UpdatePermissions
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes
	updateCmd.Flags().StringVar(&updateReq.Principal, "principal", updateReq.Principal, `Optional.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update permissions.`,
	Long: `Update permissions.
  
  Updates the permissions for a Securable type.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.SecurableType = args[0]
		updateReq.FullName = args[1]

		err = w.Grants.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Grants
