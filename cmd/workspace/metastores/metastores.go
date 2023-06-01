// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package metastores

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
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
  includes a legacy Hive metastore, the data in that metastore is available in a
  catalog named hive_metastore.`,
}

// start assign command

var assignReq catalog.CreateMetastoreAssignment
var assignJson flags.JsonFlag

func init() {
	Cmd.AddCommand(assignCmd)
	// TODO: short flags
	assignCmd.Flags().Var(&assignJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var assignCmd = &cobra.Command{
	Use:   "assign METASTORE_ID DEFAULT_CATALOG_NAME WORKSPACE_ID",
	Short: `Create an assignment.`,
	Long: `Create an assignment.
  
  Creates a new metastore assignment. If an assignment for the same
  __workspace_id__ exists, it will be overwritten by the new __metastore_id__
  and __default_catalog_name__. The caller must be an account admin.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(3)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = assignJson.Unmarshal(&assignReq)
			if err != nil {
				return err
			}
		} else {
			assignReq.MetastoreId = args[0]
			assignReq.DefaultCatalogName = args[1]
			_, err = fmt.Sscan(args[2], &assignReq.WorkspaceId)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_ID: %s", args[2])
			}
		}

		err = w.Metastores.Assign(ctx, assignReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start create command

var createReq catalog.CreateMetastore
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Region, "region", createReq.Region, `Cloud region which the metastore serves (e.g., us-west-2, westus).`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME STORAGE_ROOT",
	Short: `Create a metastore.`,
	Long: `Create a metastore.
  
  Creates a new metastore based on a provided name and storage root path.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.Name = args[0]
			createReq.StorageRoot = args[1]
		}

		response, err := w.Metastores.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start current command

func init() {
	Cmd.AddCommand(currentCmd)

}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: `Get metastore assignment for workspace.`,
	Long: `Get metastore assignment for workspace.
  
  Gets the metastore assignment for the workspace being accessed.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Metastores.Current(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteMetastoreRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	deleteCmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if the metastore is not empty.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete ID",
	Short: `Delete a metastore.`,
	Long: `Delete a metastore.
  
  Deletes a metastore. The caller must be a metastore admin.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "Loading prompts for missing command argument. You can cancel the process and provide an argument yourself instead."
				names, err := w.Metastores.MetastoreInfoNameToMetastoreIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "Unique ID of the metastore")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have unique id of the metastore")
			}
			deleteReq.Id = args[0]
		}

		err = w.Metastores.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetMetastoreRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get ID",
	Short: `Get a metastore.`,
	Long: `Get a metastore.
  
  Gets a metastore that matches the supplied ID. The caller must be a metastore
  admin to retrieve this info.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "Loading prompts for missing command argument. You can cancel the process and provide an argument yourself instead."
				names, err := w.Metastores.MetastoreInfoNameToMetastoreIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "Unique ID of the metastore")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have unique id of the metastore")
			}
			getReq.Id = args[0]
		}

		response, err := w.Metastores.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List metastores.`,
	Long: `List metastores.
  
  Gets an array of the available metastores (as __MetastoreInfo__ objects). The
  caller must be an admin to retrieve this info. There is no guarantee of a
  specific ordering of the elements in the array.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Metastores.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start maintenance command

var maintenanceReq catalog.UpdateAutoMaintenance
var maintenanceJson flags.JsonFlag

func init() {
	Cmd.AddCommand(maintenanceCmd)
	// TODO: short flags
	maintenanceCmd.Flags().Var(&maintenanceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var maintenanceCmd = &cobra.Command{
	Use:   "maintenance METASTORE_ID ENABLE",
	Short: `Enables or disables auto maintenance on the metastore.`,
	Long: `Enables or disables auto maintenance on the metastore.
  
  Enables or disables auto maintenance on the metastore.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = maintenanceJson.Unmarshal(&maintenanceReq)
			if err != nil {
				return err
			}
		} else {
			maintenanceReq.MetastoreId = args[0]
			_, err = fmt.Sscan(args[1], &maintenanceReq.Enable)
			if err != nil {
				return fmt.Errorf("invalid ENABLE: %s", args[1])
			}
		}

		response, err := w.Metastores.Maintenance(ctx, maintenanceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start summary command

func init() {
	Cmd.AddCommand(summaryCmd)

}

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: `Get a metastore summary.`,
	Long: `Get a metastore summary.
  
  Gets information about a metastore. This summary includes the storage
  credential, the cloud vendor, the cloud region, and the global metastore ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Metastores.Summary(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start unassign command

var unassignReq catalog.UnassignRequest
var unassignJson flags.JsonFlag

func init() {
	Cmd.AddCommand(unassignCmd)
	// TODO: short flags
	unassignCmd.Flags().Var(&unassignJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var unassignCmd = &cobra.Command{
	Use:   "unassign WORKSPACE_ID METASTORE_ID",
	Short: `Delete an assignment.`,
	Long: `Delete an assignment.
  
  Deletes a metastore assignment. The caller must be an account administrator.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = unassignJson.Unmarshal(&unassignReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &unassignReq.WorkspaceId)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
			}
			unassignReq.MetastoreId = args[1]
		}

		err = w.Metastores.Unassign(ctx, unassignReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq catalog.UpdateMetastore
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.DeltaSharingOrganizationName, "delta-sharing-organization-name", updateReq.DeltaSharingOrganizationName, `The organization name of a Delta Sharing entity, to be used in Databricks-to-Databricks Delta Sharing as the official name.`)
	updateCmd.Flags().Int64Var(&updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, "delta-sharing-recipient-token-lifetime-in-seconds", updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, `The lifetime of delta sharing recipient token in seconds.`)
	updateCmd.Flags().Var(&updateReq.DeltaSharingScope, "delta-sharing-scope", `The scope of Delta Sharing enabled for the metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The user-specified name of the metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The owner of the metastore.`)
	updateCmd.Flags().StringVar(&updateReq.PrivilegeModelVersion, "privilege-model-version", updateReq.PrivilegeModelVersion, `Privilege model version of the metastore, of the form major.minor (e.g., 1.0).`)
	updateCmd.Flags().StringVar(&updateReq.StorageRootCredentialId, "storage-root-credential-id", updateReq.StorageRootCredentialId, `UUID of storage credential to access the metastore storage_root.`)

}

var updateCmd = &cobra.Command{
	Use:   "update ID",
	Short: `Update a metastore.`,
	Long: `Update a metastore.
  
  Updates information for a specific metastore. The caller must be a metastore
  admin.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "Loading prompts for missing command argument. You can cancel the process and provide an argument yourself instead."
				names, err := w.Metastores.MetastoreInfoNameToMetastoreIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "Unique ID of the metastore")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have unique id of the metastore")
			}
			updateReq.Id = args[0]
		}

		response, err := w.Metastores.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update-assignment command

var updateAssignmentReq catalog.UpdateMetastoreAssignment
var updateAssignmentJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateAssignmentCmd)
	// TODO: short flags
	updateAssignmentCmd.Flags().Var(&updateAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateAssignmentCmd.Flags().StringVar(&updateAssignmentReq.DefaultCatalogName, "default-catalog-name", updateAssignmentReq.DefaultCatalogName, `The name of the default catalog for the metastore.`)
	updateAssignmentCmd.Flags().StringVar(&updateAssignmentReq.MetastoreId, "metastore-id", updateAssignmentReq.MetastoreId, `The unique ID of the metastore.`)

}

var updateAssignmentCmd = &cobra.Command{
	Use:   "update-assignment WORKSPACE_ID",
	Short: `Update an assignment.`,
	Long: `Update an assignment.
  
  Updates a metastore assignment. This operation can be used to update
  __metastore_id__ or __default_catalog_name__ for a specified Workspace, if the
  Workspace is already assigned a metastore. The caller must be an account admin
  to update __metastore_id__; otherwise, the caller can be a Workspace admin.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateAssignmentJson.Unmarshal(&updateAssignmentReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "Loading prompts for missing command argument. You can cancel the process and provide an argument yourself instead."
				names, err := w.Metastores.MetastoreInfoNameToMetastoreIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "A workspace ID")
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
		}

		err = w.Metastores.UpdateAssignment(ctx, updateAssignmentReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Metastores
