// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package metastores

import (
	"fmt"

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

// start assign command

var assignReq unitycatalog.CreateMetastoreAssignment

func init() {
	Cmd.AddCommand(assignCmd)
	// TODO: short flags

}

var assignCmd = &cobra.Command{
	Use:   "assign METASTORE_ID DEFAULT_CATALOG_NAME WORKSPACE_ID",
	Short: `Create an assignment.`,
	Long: `Create an assignment.
  
  Creates a new Metastore assignment. If an assignment for the same
  __workspace_id__ exists, it will be overwritten by the new __metastore_id__
  and __default_catalog_name__. The caller must be an account admin.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		assignReq.MetastoreId = args[0]
		assignReq.DefaultCatalogName = args[1]
		_, err = fmt.Sscan(args[2], &assignReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[2])
		}

		err = w.Metastores.Assign(ctx, assignReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start create command

var createReq unitycatalog.CreateMetastore

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

}

var createCmd = &cobra.Command{
	Use:   "create NAME STORAGE_ROOT",
	Short: `Create a Metastore.`,
	Long: `Create a Metastore.
  
  Creates a new Metastore based on a provided name and storage root path.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		createReq.Name = args[0]
		createReq.StorageRoot = args[1]

		response, err := w.Metastores.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq unitycatalog.DeleteMetastoreRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if the metastore is not empty.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete ID",
	Short: `Delete a Metastore.`,
	Long: `Delete a Metastore.
  
  Deletes a Metastore. The caller must be a Metastore admin.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Metastores.MetastoreInfoNameToMetastoreIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		deleteReq.Id = args[0]

		err = w.Metastores.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq unitycatalog.GetMetastoreRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get ID",
	Short: `Get a Metastore.`,
	Long: `Get a Metastore.
  
  Gets a Metastore that matches the supplied ID. The caller must be a Metastore
  admin to retrieve this info.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Metastores.MetastoreInfoNameToMetastoreIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		getReq.Id = args[0]

		response, err := w.Metastores.Get(ctx, getReq)
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
	Short: `List Metastores.`,
	Long: `List Metastores.
  
  Gets an array of the available Metastores (as MetastoreInfo objects). The
  caller must be an admin to retrieve this info.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Metastores.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start summary command

func init() {
	Cmd.AddCommand(summaryCmd)

}

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: `Get a summary.`,
	Long: `Get a summary.
  
  Gets information about a Metastore. This summary includes the storage
  credential, the cloud vendor, the cloud region, and the global Metastore ID.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Metastores.Summary(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start unassign command

var unassignReq unitycatalog.UnassignRequest

func init() {
	Cmd.AddCommand(unassignCmd)
	// TODO: short flags

}

var unassignCmd = &cobra.Command{
	Use:   "unassign WORKSPACE_ID METASTORE_ID",
	Short: `Delete an assignment.`,
	Long: `Delete an assignment.
  
  Deletes a Metastore assignment. The caller must be an account administrator.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		_, err = fmt.Sscan(args[0], &unassignReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		unassignReq.MetastoreId = args[1]

		err = w.Metastores.Unassign(ctx, unassignReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq unitycatalog.UpdateMetastore

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.DefaultDataAccessConfigId, "default-data-access-config-id", updateReq.DefaultDataAccessConfigId, `Unique identifier of (Default) Data Access Configuration.`)
	updateCmd.Flags().BoolVar(&updateReq.DeltaSharingEnabled, "delta-sharing-enabled", updateReq.DeltaSharingEnabled, `Whether Delta Sharing is enabled on this metastore.`)
	updateCmd.Flags().IntVar(&updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, "delta-sharing-recipient-token-lifetime-in-seconds", updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, `The lifetime of delta sharing recipient token in seconds.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of Metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The owner of the metastore.`)
	updateCmd.Flags().StringVar(&updateReq.StorageRootCredentialId, "storage-root-credential-id", updateReq.StorageRootCredentialId, `UUID of storage credential to access storage_root.`)

}

var updateCmd = &cobra.Command{
	Use:   "update ID",
	Short: `Update a Metastore.`,
	Long: `Update a Metastore.
  
  Updates information for a specific Metastore. The caller must be a Metastore
  admin.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Metastores.MetastoreInfoNameToMetastoreIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		updateReq.Id = args[0]

		err = w.Metastores.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update-assignment command

var updateAssignmentReq unitycatalog.UpdateMetastoreAssignment

func init() {
	Cmd.AddCommand(updateAssignmentCmd)
	// TODO: short flags

	updateAssignmentCmd.Flags().StringVar(&updateAssignmentReq.DefaultCatalogName, "default-catalog-name", updateAssignmentReq.DefaultCatalogName, `The name of the default catalog for the Metastore.`)
	updateAssignmentCmd.Flags().StringVar(&updateAssignmentReq.MetastoreId, "metastore-id", updateAssignmentReq.MetastoreId, `The unique ID of the Metastore.`)

}

var updateAssignmentCmd = &cobra.Command{
	Use:   "update-assignment WORKSPACE_ID",
	Short: `Update an assignment.`,
	Long: `Update an assignment.
  
  Updates a Metastore assignment. This operation can be used to update
  __metastore_id__ or __default_catalog_name__ for a specified Workspace, if the
  Workspace is already assigned a Metastore. The caller must be an account admin
  to update __metastore_id__; otherwise, the caller can be a Workspace admin.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Metastores.MetastoreInfoNameToMetastoreIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "A workspace ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have a workspace id")
		}
		_, err = fmt.Sscan(args[0], &updateAssignmentReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		err = w.Metastores.UpdateAssignment(ctx, updateAssignmentReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Metastores
