package metastores

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "metastores",
	Short: `A metastore is the top-level container of objects in Unity Catalog.`,
	Long: `A metastore is the top-level container of objects in Unity Catalog. It stores
  data assets (tables and views) and the permissions that govern access to them.
  Databricks account admins can create metastores and assign them to Databricks
  workspaces to control which workloads use each metastore. For a workspace to
  use Unity Catalog, it must have a Unity Catalog metastore attached.
  
  Each metastore is configured with a root storage location in a cloud storage
  account. This storage location is used for metadata and managed tables data.
  
  NOTE: This metastore is distinct from the metastore included in Databricks
  workspaces created before Unity Catalog was released. If your workspace
  includes a legacy Hive metastore, the data in that metastore is available in
  Unity Catalog in a catalog named hive_metastore.`,
}

var assignReq unitycatalog.CreateMetastoreAssignment

func init() {
	Cmd.AddCommand(assignCmd)
	// TODO: short flags

	assignCmd.Flags().StringVar(&assignReq.DefaultCatalogName, "default-catalog-name", "", `THe name of the default catalog in the Metastore.`)
	assignCmd.Flags().StringVar(&assignReq.MetastoreId, "metastore-id", "", `The ID of the Metastore.`)
	assignCmd.Flags().IntVar(&assignReq.WorkspaceId, "workspace-id", 0, `A workspace ID.`)

}

var assignCmd = &cobra.Command{
	Use:   "assign",
	Short: `Create an assignment.`,
	Long: `Create an assignment.
  
  Creates a new Metastore assignment. If an assignment for the same
  __workspace_id__ exists, it will be overwritten by the new __metastore_id__
  and __default_catalog_name__. The caller must be an account admin.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Metastores.Assign(ctx, assignReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var createReq unitycatalog.CreateMetastore

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().Int64Var(&createReq.CreatedAt, "created-at", 0, `[Create,Update:IGN] Time at which this Metastore was created, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.CreatedBy, "created-by", "", `[Create,Update:IGN] Username of Metastore creator.`)
	createCmd.Flags().StringVar(&createReq.DefaultDataAccessConfigId, "default-data-access-config-id", "", `[Create:IGN Update:OPT] Unique identifier of (Default) Data Access Configuration.`)
	createCmd.Flags().BoolVar(&createReq.DeltaSharingEnabled, "delta-sharing-enabled", false, `[Create:IGN Update:OPT] Whether Delta Sharing is enabled on this metastore.`)
	createCmd.Flags().IntVar(&createReq.DeltaSharingRecipientTokenLifetimeInSeconds, "delta-sharing-recipient-token-lifetime-in-seconds", 0, `[Create:IGN Update:OPT] The lifetime of delta sharing recipient token in seconds.`)
	createCmd.Flags().StringVar(&createReq.MetastoreId, "metastore-id", "", `[Create,Update:IGN] Unique identifier of Metastore.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `[Create:REQ Update:OPT] Name of Metastore.`)
	createCmd.Flags().StringVar(&createReq.Owner, "owner", "", `[Create:IGN Update:OPT] The owner of the metastore.`)
	// TODO: array: privileges
	createCmd.Flags().StringVar(&createReq.Region, "region", "", `The region this metastore has an afinity to.`)
	createCmd.Flags().StringVar(&createReq.StorageRoot, "storage-root", "", `[Create:REQ Update:ERR] Storage root URL for Metastore.`)
	createCmd.Flags().StringVar(&createReq.StorageRootCredentialId, "storage-root-credential-id", "", `[Create:IGN Update:OPT] UUID of storage credential to access storage_root.`)
	createCmd.Flags().Int64Var(&createReq.UpdatedAt, "updated-at", 0, `[Create,Update:IGN] Time at which the Metastore was last modified, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.UpdatedBy, "updated-by", "", `[Create,Update:IGN] Username of user who last modified the Metastore.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a Metastore.`,
	Long: `Create a Metastore.
  
  Creates a new Metastore based on a provided name and storage root path.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Metastores.Create(ctx, createReq)
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

var deleteReq unitycatalog.DeleteMetastoreRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().BoolVar(&deleteReq.Force, "force", false, `Force deletion even if the metastore is not empty.`)
	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", "", `Required.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a Metastore.`,
	Long: `Delete a Metastore.
  
  Deletes a Metastore. The caller must be a Metastore admin.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Metastores.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq unitycatalog.GetMetastoreRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Id, "id", "", `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a Metastore.`,
	Long: `Get a Metastore.
  
  Gets a Metastore that matches the supplied ID. The caller must be a Metastore
  admin to retrieve this info.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Metastores.Get(ctx, getReq)
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
	Short: `List Metastores.`,
	Long: `List Metastores.
  
  Gets an array of the available Metastores (as MetastoreInfo objects). The
  caller must be an admin to retrieve this info.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Metastores.ListAll(ctx)
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
	Cmd.AddCommand(summaryCmd)

}

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: `Get a summary.`,
	Long: `Get a summary.
  
  Gets information about a Metastore. This summary includes the storage
  credential, the cloud vendor, the cloud region, and the global Metastore ID.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Metastores.Summary(ctx)
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

var unassignReq unitycatalog.UnassignRequest

func init() {
	Cmd.AddCommand(unassignCmd)
	// TODO: short flags

	unassignCmd.Flags().StringVar(&unassignReq.MetastoreId, "metastore-id", "", `Query for the ID of the Metastore to delete.`)
	unassignCmd.Flags().IntVar(&unassignReq.WorkspaceId, "workspace-id", 0, `A workspace ID.`)

}

var unassignCmd = &cobra.Command{
	Use:   "unassign",
	Short: `Delete an assignment.`,
	Long: `Delete an assignment.
  
  Deletes a Metastore assignment. The caller must be an account administrator.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Metastores.Unassign(ctx, unassignReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateReq unitycatalog.UpdateMetastore

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().Int64Var(&updateReq.CreatedAt, "created-at", 0, `[Create,Update:IGN] Time at which this Metastore was created, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.CreatedBy, "created-by", "", `[Create,Update:IGN] Username of Metastore creator.`)
	updateCmd.Flags().StringVar(&updateReq.DefaultDataAccessConfigId, "default-data-access-config-id", "", `[Create:IGN Update:OPT] Unique identifier of (Default) Data Access Configuration.`)
	updateCmd.Flags().BoolVar(&updateReq.DeltaSharingEnabled, "delta-sharing-enabled", false, `[Create:IGN Update:OPT] Whether Delta Sharing is enabled on this metastore.`)
	updateCmd.Flags().IntVar(&updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, "delta-sharing-recipient-token-lifetime-in-seconds", 0, `[Create:IGN Update:OPT] The lifetime of delta sharing recipient token in seconds.`)
	updateCmd.Flags().StringVar(&updateReq.Id, "id", "", `Required.`)
	updateCmd.Flags().StringVar(&updateReq.MetastoreId, "metastore-id", "", `[Create,Update:IGN] Unique identifier of Metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", "", `[Create:REQ Update:OPT] Name of Metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", "", `[Create:IGN Update:OPT] The owner of the metastore.`)
	// TODO: array: privileges
	updateCmd.Flags().StringVar(&updateReq.Region, "region", "", `The region this metastore has an afinity to.`)
	updateCmd.Flags().StringVar(&updateReq.StorageRoot, "storage-root", "", `[Create:REQ Update:ERR] Storage root URL for Metastore.`)
	updateCmd.Flags().StringVar(&updateReq.StorageRootCredentialId, "storage-root-credential-id", "", `[Create:IGN Update:OPT] UUID of storage credential to access storage_root.`)
	updateCmd.Flags().Int64Var(&updateReq.UpdatedAt, "updated-at", 0, `[Create,Update:IGN] Time at which the Metastore was last modified, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.UpdatedBy, "updated-by", "", `[Create,Update:IGN] Username of user who last modified the Metastore.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a Metastore.`,
	Long: `Update a Metastore.
  
  Updates information for a specific Metastore. The caller must be a Metastore
  admin.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Metastores.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateAssignmentReq unitycatalog.UpdateMetastoreAssignment

func init() {
	Cmd.AddCommand(updateAssignmentCmd)
	// TODO: short flags

	updateAssignmentCmd.Flags().StringVar(&updateAssignmentReq.DefaultCatalogName, "default-catalog-name", "", `The name of the default catalog for the Metastore.`)
	updateAssignmentCmd.Flags().StringVar(&updateAssignmentReq.MetastoreId, "metastore-id", "", `The unique ID of the Metastore.`)
	updateAssignmentCmd.Flags().IntVar(&updateAssignmentReq.WorkspaceId, "workspace-id", 0, `A workspace ID.`)

}

var updateAssignmentCmd = &cobra.Command{
	Use:   "update-assignment",
	Short: `Update an assignment.`,
	Long: `Update an assignment.
  
  Updates a Metastore assignment. This operation can be used to update
  __metastore_id__ or __default_catalog_name__ for a specified Workspace, if the
  Workspace is already assigned a Metastore. The caller must be an account admin
  to update __metastore_id__; otherwise, the caller can be a Workspace admin.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Metastores.UpdateAssignment(ctx, updateAssignmentReq)
		if err != nil {
			return err
		}

		return nil
	},
}

// end service Metastores
