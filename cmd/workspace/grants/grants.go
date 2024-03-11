// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package grants

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
		Use:   "grants",
		Short: `In Unity Catalog, data is secure by default.`,
		Long: `In Unity Catalog, data is secure by default. Initially, users have no access
  to data in a metastore. Access can be granted by either a metastore admin, the
  owner of an object, or the owner of the catalog or schema that contains the
  object. Securable objects in Unity Catalog are hierarchical and privileges are
  inherited downward.
  
  Securable objects in Unity Catalog are hierarchical and privileges are
  inherited downward. This means that granting a privilege on the catalog
  automatically grants the privilege to all current and future objects within
  the catalog. Similarly, privileges granted on a schema are inherited by all
  current and future objects within that schema.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
	}

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetEffective())
	cmd.AddCommand(newUpdate())

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
	*catalog.GetGrantRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetGrantRequest

	// TODO: short flags

	cmd.Flags().StringVar(&getReq.Principal, "principal", getReq.Principal, `If provided, only the permissions for the specified principal (user or group) are returned.`)

	cmd.Use = "get SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Get permissions.`
	cmd.Long = `Get permissions.
  
  Gets the permissions for a securable.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

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

		_, err = fmt.Sscan(args[0], &getReq.SecurableType)
		if err != nil {
			return fmt.Errorf("invalid SECURABLE_TYPE: %s", args[0])
		}
		getReq.FullName = args[1]

		response, err := w.Grants.Get(ctx, getReq)
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

// start get-effective command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEffectiveOverrides []func(
	*cobra.Command,
	*catalog.GetEffectiveRequest,
)

func newGetEffective() *cobra.Command {
	cmd := &cobra.Command{}

	var getEffectiveReq catalog.GetEffectiveRequest

	// TODO: short flags

	cmd.Flags().StringVar(&getEffectiveReq.Principal, "principal", getEffectiveReq.Principal, `If provided, only the effective permissions for the specified principal (user or group) are returned.`)

	cmd.Use = "get-effective SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Get effective permissions.`
	cmd.Long = `Get effective permissions.
  
  Gets the effective permissions for a securable.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

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

		_, err = fmt.Sscan(args[0], &getEffectiveReq.SecurableType)
		if err != nil {
			return fmt.Errorf("invalid SECURABLE_TYPE: %s", args[0])
		}
		getEffectiveReq.FullName = args[1]

		response, err := w.Grants.GetEffective(ctx, getEffectiveReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEffectiveOverrides {
		fn(cmd, &getEffectiveReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdatePermissions,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdatePermissions
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes

	cmd.Use = "update SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Update permissions.`
	cmd.Long = `Update permissions.
  
  Updates the permissions for a securable.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

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
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		}
		_, err = fmt.Sscan(args[0], &updateReq.SecurableType)
		if err != nil {
			return fmt.Errorf("invalid SECURABLE_TYPE: %s", args[0])
		}
		updateReq.FullName = args[1]

		response, err := w.Grants.Update(ctx, updateReq)
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

// end service Grants
