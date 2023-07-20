// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_bindings

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-bindings",
		Short: `A catalog in Databricks can be configured as __OPEN__ or __ISOLATED__.`,
		Long: `A catalog in Databricks can be configured as __OPEN__ or __ISOLATED__. An
  __OPEN__ catalog can be accessed from any workspace, while an __ISOLATED__
  catalog can only be access from a configured list of workspaces.
  
  A catalog's workspace bindings can be configured by a metastore admin or the
  owner of the catalog.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
	}

	cmd.AddCommand(newGet())
	cmd.AddCommand(newUpdate())

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*catalog.GetWorkspaceBindingRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetWorkspaceBindingRequest

	// TODO: short flags

	cmd.Use = "get NAME"
	cmd.Short = `Get catalog workspace bindings.`
	cmd.Long = `Get catalog workspace bindings.
  
  Gets workspace bindings of the catalog. The caller must be a metastore admin
  or an owner of the catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.Name = args[0]

		response, err := w.WorkspaceBindings.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateWorkspaceBindings,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateWorkspaceBindings
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: assign_workspaces
	// TODO: array: unassign_workspaces

	cmd.Use = "update NAME"
	cmd.Short = `Update catalog workspace bindings.`
	cmd.Long = `Update catalog workspace bindings.
  
  Updates workspace bindings of the catalog. The caller must be a metastore
  admin or an owner of the catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		}
		updateReq.Name = args[0]

		response, err := w.WorkspaceBindings.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service WorkspaceBindings
