// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_assignment

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace-assignment",
	Short: `The Workspace Permission Assignment API allows you to manage workspace permissions for principals in your account.`,
	Long: `The Workspace Permission Assignment API allows you to manage workspace
  permissions for principals in your account.`,
	Annotations: map[string]string{
		"package": "iam",
	},
}

// start delete command
var deleteReq iam.DeleteWorkspaceAssignmentRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete WORKSPACE_ID PRINCIPAL_ID",
	Short: `Delete permissions assignment.`,
	Long: `Delete permissions assignment.
  
  Deletes the workspace permissions assignment in a given account and workspace
  for the specified principal.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &deleteReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		err = a.WorkspaceAssignment.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command
var getReq iam.GetWorkspaceAssignmentRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get WORKSPACE_ID",
	Short: `List workspace permissions.`,
	Long: `List workspace permissions.
  
  Get an array of workspace permissions for the specified account and workspace.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &getReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response, err := a.WorkspaceAssignment.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command
var listReq iam.ListWorkspaceAssignmentRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

}

var listCmd = &cobra.Command{
	Use:   "list WORKSPACE_ID",
	Short: `Get permission assignments.`,
	Long: `Get permission assignments.
  
  Get the permission assignments for the specified Databricks account and
  Databricks workspace.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &listReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response, err := a.WorkspaceAssignment.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start update command
var updateReq iam.UpdateWorkspaceAssignments
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Create or update permissions assignment.`,
	Long: `Create or update permissions assignment.
  
  Creates or updates the workspace permissions assignment in a given account and
  workspace for the specified principal.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = a.WorkspaceAssignment.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service WorkspaceAssignment
