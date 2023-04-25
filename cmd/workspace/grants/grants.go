// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package grants

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
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
  
  Securable objects in Unity Catalog are hierarchical and privileges are
  inherited downward. This means that granting a privilege on the catalog
  automatically grants the privilege to all current and future objects within
  the catalog. Similarly, privileges granted on a schema are inherited by all
  current and future objects within that schema.`,
}

// start get command

var getReq catalog.GetGrantRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Principal, "principal", getReq.Principal, `If provided, only the permissions for the specified principal (user or group) are returned.`)

}

var getCmd = &cobra.Command{
	Use:   "get SECURABLE_TYPE FULL_NAME",
	Short: `Get permissions.`,
	Long: `Get permissions.
  
  Gets the permissions for a securable.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		_, err = fmt.Sscan(args[0], &getReq.SecurableType)
		if err != nil {
			return fmt.Errorf("invalid SECURABLE_TYPE: %s", args[0])
		}
		getReq.FullName = args[1]

		response, err := w.Grants.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start get-effective command

var getEffectiveReq catalog.GetEffectiveRequest

func init() {
	Cmd.AddCommand(getEffectiveCmd)
	// TODO: short flags

	getEffectiveCmd.Flags().StringVar(&getEffectiveReq.Principal, "principal", getEffectiveReq.Principal, `If provided, only the effective permissions for the specified principal (user or group) are returned.`)

}

var getEffectiveCmd = &cobra.Command{
	Use:   "get-effective SECURABLE_TYPE FULL_NAME",
	Short: `Get effective permissions.`,
	Long: `Get effective permissions.
  
  Gets the effective permissions for a securable.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		_, err = fmt.Sscan(args[0], &getEffectiveReq.SecurableType)
		if err != nil {
			return fmt.Errorf("invalid SECURABLE_TYPE: %s", args[0])
		}
		getEffectiveReq.FullName = args[1]

		response, err := w.Grants.GetEffective(ctx, getEffectiveReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.UpdatePermissions
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update permissions.`,
	Long: `Update permissions.
  
  Updates the permissions for a securable.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		_, err = fmt.Sscan(args[0], &updateReq.SecurableType)
		if err != nil {
			return fmt.Errorf("invalid SECURABLE_TYPE: %s", args[0])
		}
		updateReq.FullName = args[1]

		response, err := w.Grants.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service Grants
