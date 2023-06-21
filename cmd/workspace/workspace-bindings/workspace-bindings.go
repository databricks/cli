// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_bindings

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace-bindings",
	Short: `A catalog in Databricks can be configured as __OPEN__ or __ISOLATED__.`,
	Long: `A catalog in Databricks can be configured as __OPEN__ or __ISOLATED__. An
  __OPEN__ catalog can be accessed from any workspace, while an __ISOLATED__
  catalog can only be access from a configured list of workspaces.
  
  A catalog's workspace bindings can be configured by a metastore admin or the
  owner of the catalog.`,
	Annotations: map[string]string{
		"package": "catalog",
	},
}

// start get command

var getReq catalog.GetWorkspaceBindingRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get catalog workspace bindings.`,
	Long: `Get catalog workspace bindings.
  
  Gets workspace bindings of the catalog. The caller must be a metastore admin
  or an owner of the catalog.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
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
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			getReq.Name = args[0]
		}

		response, err := w.WorkspaceBindings.Get(ctx, getReq)
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

var updateReq catalog.UpdateWorkspaceBindings
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: assign_workspaces
	// TODO: array: unassign_workspaces

}

var updateCmd = &cobra.Command{
	Use:   "update NAME",
	Short: `Update catalog workspace bindings.`,
	Long: `Update catalog workspace bindings.
  
  Updates workspace bindings of the catalog. The caller must be a metastore
  admin or an owner of the catalog.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
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
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			updateReq.Name = args[0]
		}

		response, err := w.WorkspaceBindings.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service WorkspaceBindings
