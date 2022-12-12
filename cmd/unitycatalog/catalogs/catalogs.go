// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package catalogs

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "catalogs",
	Short: `A catalog is the first layer of Unity Catalog’s three-level namespace.`,
	Long: `A catalog is the first layer of Unity Catalog’s three-level namespace.
  It’s used to organize your data assets. Users can see all catalogs on which
  they have been assigned the USAGE data permission.
  
  In Unity Catalog, admins and data stewards manage users and their access to
  data centrally across all of the workspaces in a Databricks account. Users in
  different workspaces can share access to the same data, depending on
  privileges granted centrally in Unity Catalog.`,
}

// start create command

var createReq unitycatalog.CreateCatalog
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	// TODO: map via StringToStringVar: properties
	createCmd.Flags().StringVar(&createReq.ProviderName, "provider-name", createReq.ProviderName, `The name of delta sharing provider.`)
	createCmd.Flags().StringVar(&createReq.ShareName, "share-name", createReq.ShareName, `The name of the share under the share provider.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a catalog.`,
	Long: `Create a catalog.
  
  Creates a new catalog instance in the parent Metastore if the caller is a
  Metastore admin or has the CREATE CATALOG privilege.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]

		response, err := w.Catalogs.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq unitycatalog.DeleteCatalogRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete a catalog.`,
	Long: `Delete a catalog.
  
  Deletes the catalog that matches the supplied name. The caller must be a
  Metastore admin or the owner of the catalog.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteReq.Name = args[0]

		err = w.Catalogs.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq unitycatalog.GetCatalogRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get a catalog.`,
	Long: `Get a catalog.
  
  Gets an array of all catalogs in the current Metastore for which the user is
  an admin or Catalog owner, or has the USAGE privilege set for their account.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getReq.Name = args[0]

		response, err := w.Catalogs.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List catalogs.`,
	Long: `List catalogs.
  
  Gets an array of External Locations (ExternalLocationInfo objects) from the
  Metastore. The caller must be a Metastore admin, is the owner of the External
  Location, or has privileges to access the External Location.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Catalogs.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq unitycatalog.UpdateCatalog
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of Catalog.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of Catalog.`)
	// TODO: map via StringToStringVar: properties

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a catalog.`,
	Long: `Update a catalog.
  
  Updates the catalog that matches the supplied name. The caller must be either
  the owner of the catalog, or a Metastore admin (when changing the owner field
  of the catalog).`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.Name = args[0]

		err = w.Catalogs.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Catalogs
