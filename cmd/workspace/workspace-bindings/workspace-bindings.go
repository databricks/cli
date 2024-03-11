// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_bindings

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-bindings",
		Short: `A securable in Databricks can be configured as __OPEN__ or __ISOLATED__.`,
		Long: `A securable in Databricks can be configured as __OPEN__ or __ISOLATED__. An
  __OPEN__ securable can be accessed from any workspace, while an __ISOLATED__
  securable can only be accessed from a configured list of workspaces. This API
  allows you to configure (bind) securables to workspaces.
  
  NOTE: The __isolation_mode__ is configured for the securable itself (using its
  Update method) and the workspace bindings are only consulted when the
  securable's __isolation_mode__ is set to __ISOLATED__.
  
  A securable's workspace bindings can be configured by a metastore admin or the
  owner of the securable.
  
  The original path (/api/2.1/unity-catalog/workspace-bindings/catalogs/{name})
  is deprecated. Please use the new path
  (/api/2.1/unity-catalog/bindings/{securable_type}/{securable_name}) which
  introduces the ability to bind a securable in READ_ONLY mode (catalogs only).
  
  Securables that support binding: - catalog`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
	}

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetBindings())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdateBindings())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

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
  or an owner of the catalog.

  Arguments:
    NAME: The name of the catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
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

// start get-bindings command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getBindingsOverrides []func(
	*cobra.Command,
	*catalog.GetBindingsRequest,
)

func newGetBindings() *cobra.Command {
	cmd := &cobra.Command{}

	var getBindingsReq catalog.GetBindingsRequest

	// TODO: short flags

	cmd.Use = "get-bindings SECURABLE_TYPE SECURABLE_NAME"
	cmd.Short = `Get securable workspace bindings.`
	cmd.Long = `Get securable workspace bindings.
  
  Gets workspace bindings of the securable. The caller must be a metastore admin
  or an owner of the securable.

  Arguments:
    SECURABLE_TYPE: The type of the securable.
    SECURABLE_NAME: The name of the securable.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getBindingsReq.SecurableType = args[0]
		getBindingsReq.SecurableName = args[1]

		response, err := w.WorkspaceBindings.GetBindings(ctx, getBindingsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getBindingsOverrides {
		fn(cmd, &getBindingsReq)
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
  admin or an owner of the catalog.

  Arguments:
    NAME: The name of the catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
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

// start update-bindings command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateBindingsOverrides []func(
	*cobra.Command,
	*catalog.UpdateWorkspaceBindingsParameters,
)

func newUpdateBindings() *cobra.Command {
	cmd := &cobra.Command{}

	var updateBindingsReq catalog.UpdateWorkspaceBindingsParameters
	var updateBindingsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateBindingsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: add
	// TODO: array: remove

	cmd.Use = "update-bindings SECURABLE_TYPE SECURABLE_NAME"
	cmd.Short = `Update securable workspace bindings.`
	cmd.Long = `Update securable workspace bindings.
  
  Updates workspace bindings of the securable. The caller must be a metastore
  admin or an owner of the securable.

  Arguments:
    SECURABLE_TYPE: The type of the securable.
    SECURABLE_NAME: The name of the securable.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateBindingsJson.Unmarshal(&updateBindingsReq)
			if err != nil {
				return err
			}
		}
		updateBindingsReq.SecurableType = args[0]
		updateBindingsReq.SecurableName = args[1]

		response, err := w.WorkspaceBindings.UpdateBindings(ctx, updateBindingsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateBindingsOverrides {
		fn(cmd, &updateBindingsReq)
	}

	return cmd
}

// end service WorkspaceBindings
