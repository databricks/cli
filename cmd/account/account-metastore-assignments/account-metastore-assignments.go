// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package account_metastore_assignments

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "account-metastore-assignments",
	Short: `These APIs manage metastore assignments to a workspace.`,
	Long:  `These APIs manage metastore assignments to a workspace.`,
}

// start create command

var createReq catalog.CreateMetastoreAssignment

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

}

var createCmd = &cobra.Command{
	Use:   "create METASTORE_ID DEFAULT_CATALOG_NAME WORKSPACE_ID",
	Short: `Assigns a workspace to a metastore.`,
	Long: `Assigns a workspace to a metastore.
  
  Creates an assignment to a metastore for a workspace`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		createReq.MetastoreId = args[0]
		createReq.DefaultCatalogName = args[1]
		_, err = fmt.Sscan(args[2], &createReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[2])
		}

		response, err := a.AccountMetastoreAssignments.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteAccountMetastoreAssignmentRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete WORKSPACE_ID METASTORE_ID",
	Short: `Delete a metastore assignment.`,
	Long: `Delete a metastore assignment.
  
  Deletes a metastore assignment to a workspace, leaving the workspace with no
  metastore.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		_, err = fmt.Sscan(args[0], &deleteReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		deleteReq.MetastoreId = args[1]

		err = a.AccountMetastoreAssignments.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetAccountMetastoreAssignmentRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get WORKSPACE_ID",
	Short: `Gets the metastore assignment for a workspace.`,
	Long: `Gets the metastore assignment for a workspace.
  
  Gets the metastore assignment, if any, for the workspace specified by ID. If
  the workspace is assigned a metastore, the mappig will be returned. If no
  metastore is assigned to the workspace, the assignment will not be found and a
  404 returned.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		_, err = fmt.Sscan(args[0], &getReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response, err := a.AccountMetastoreAssignments.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq catalog.ListAccountMetastoreAssignmentsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

}

var listCmd = &cobra.Command{
	Use:   "list METASTORE_ID",
	Short: `Get all workspaces assigned to a metastore.`,
	Long: `Get all workspaces assigned to a metastore.
  
  Gets a list of all Databricks workspace IDs that have been assigned to given
  metastore.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		listReq.MetastoreId = args[0]

		response, err := a.AccountMetastoreAssignments.List(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.UpdateMetastoreAssignment

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.DefaultCatalogName, "default-catalog-name", updateReq.DefaultCatalogName, `The name of the default catalog for the metastore.`)
	updateCmd.Flags().StringVar(&updateReq.MetastoreId, "metastore-id", updateReq.MetastoreId, `The unique ID of the metastore.`)

}

var updateCmd = &cobra.Command{
	Use:   "update WORKSPACE_ID METASTORE_ID",
	Short: `Updates a metastore assignment to a workspaces.`,
	Long: `Updates a metastore assignment to a workspaces.
  
  Updates an assignment to a metastore for a workspace. Currently, only the
  default catalog may be updated`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		_, err = fmt.Sscan(args[0], &updateReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		updateReq.MetastoreId = args[1]

		response, err := a.AccountMetastoreAssignments.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service AccountMetastoreAssignments
