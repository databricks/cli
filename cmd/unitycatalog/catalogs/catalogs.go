package catalogs

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "catalogs",
	Short: `A catalog is the first layer of Unity Catalog’s three-level namespace.`,
}

var createReq unitycatalog.CreateCatalog

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", "", `User-provided free-form text description.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `Name of Catalog.`)
	// TODO: array: properties
	createCmd.Flags().StringVar(&createReq.ProviderName, "provider-name", "", `Delta Sharing Catalog specific fields.`)
	createCmd.Flags().StringVar(&createReq.ShareName, "share-name", "", `The name of the share under the share provider.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a catalog.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Catalogs.Create(ctx, createReq)
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

var deleteReq unitycatalog.DeleteCatalogRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", "", `Required.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a catalog.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Catalogs.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq unitycatalog.GetCatalogRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Name, "name", "", `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a catalog.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Catalogs.Get(ctx, getReq)
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

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List catalogs.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Catalogs.ListAll(ctx)
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

var updateReq unitycatalog.UpdateCatalog

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().Var(&updateReq.CatalogType, "catalog-type", `[Create,Update:IGN] The type of the catalog.`)
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", "", `[Create,Update:OPT] User-provided free-form text description.`)
	updateCmd.Flags().Int64Var(&updateReq.CreatedAt, "created-at", 0, `[Create,Update:IGN] Time at which this Catalog was created, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.CreatedBy, "created-by", "", `[Create,Update:IGN] Username of Catalog creator.`)
	updateCmd.Flags().StringVar(&updateReq.MetastoreId, "metastore-id", "", `[Create,Update:IGN] Unique identifier of parent Metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", "", `[Create:REQ Update:OPT] Name of Catalog.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", "", `[Create:IGN,Update:OPT] Username of current owner of Catalog.`)
	// TODO: array: privileges
	// TODO: array: properties
	updateCmd.Flags().StringVar(&updateReq.ProviderName, "provider-name", "", `Delta Sharing Catalog specific fields.`)
	updateCmd.Flags().StringVar(&updateReq.ShareName, "share-name", "", `[Create:OPT,Update: IGN] The name of the share under the share provider.`)
	updateCmd.Flags().Int64Var(&updateReq.UpdatedAt, "updated-at", 0, `[Create,Update:IGN] Time at which this Catalog was last modified, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.UpdatedBy, "updated-by", "", `[Create,Update:IGN] Username of user who last modified Catalog.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a catalog.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Catalogs.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
