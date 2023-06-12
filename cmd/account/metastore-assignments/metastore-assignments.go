// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package metastore_assignments

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "metastore-assignments",
	Short: `These APIs manage metastore assignments to a workspace.`,
	Long:  `These APIs manage metastore assignments to a workspace.`,
}

// start create command

var createReq catalog.AccountsCreateMetastoreAssignment
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: metastore_assignment

}

var createCmd = &cobra.Command{
	Use:   "create WORKSPACE_ID METASTORE_ID",
	Short: `Assigns a workspace to a metastore.`,
	Long: `Assigns a workspace to a metastore.
  
  Creates an assignment to a metastore for a workspace Please add a header
  X-Databricks-Account-Console-API-Version: 2.0 to access this API.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &createReq.WorkspaceId)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
			}
			createReq.MetastoreId = args[1]
		}

		response, err := a.MetastoreAssignments.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteAccountMetastoreAssignmentRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete WORKSPACE_ID METASTORE_ID",
	Short: `Delete a metastore assignment.`,
	Long: `Delete a metastore assignment.
  
  Deletes a metastore assignment to a workspace, leaving the workspace with no
  metastore. Please add a header X-Databricks-Account-Console-API-Version: 2.0
  to access this API.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &deleteReq.WorkspaceId)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
			}
			deleteReq.MetastoreId = args[1]
		}

		err = a.MetastoreAssignments.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetAccountMetastoreAssignmentRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get WORKSPACE_ID",
	Short: `Gets the metastore assignment for a workspace.`,
	Long: `Gets the metastore assignment for a workspace.
  
  Gets the metastore assignment, if any, for the workspace specified by ID. If
  the workspace is assigned a metastore, the mappig will be returned. If no
  metastore is assigned to the workspace, the assignment will not be found and a
  404 returned. Please add a header X-Databricks-Account-Console-API-Version:
  2.0 to access this API.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &getReq.WorkspaceId)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
			}
		}

		response, err := a.MetastoreAssignments.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq catalog.ListAccountMetastoreAssignmentsRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var listCmd = &cobra.Command{
	Use:   "list METASTORE_ID",
	Short: `Get all workspaces assigned to a metastore.`,
	Long: `Get all workspaces assigned to a metastore.
  
  Gets a list of all Databricks workspace IDs that have been assigned to given
  metastore. Please add a header X-Databricks-Account-Console-API-Version: 2.0
  to access this API`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
			listReq.MetastoreId = args[0]
		}

		response, err := a.MetastoreAssignments.List(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.AccountsUpdateMetastoreAssignment
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: metastore_assignment

}

var updateCmd = &cobra.Command{
	Use:   "update WORKSPACE_ID METASTORE_ID",
	Short: `Updates a metastore assignment to a workspaces.`,
	Long: `Updates a metastore assignment to a workspaces.
  
  Updates an assignment to a metastore for a workspace. Currently, only the
  default catalog may be updated. Please add a header
  X-Databricks-Account-Console-API-Version: 2.0 to access this API.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &updateReq.WorkspaceId)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
			}
			updateReq.MetastoreId = args[1]
		}

		response, err := a.MetastoreAssignments.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service AccountMetastoreAssignments
