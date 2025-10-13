// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package grants

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		RunE: root.ReportUnknownSubcommand,
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

	cmd.Flags().IntVar(&getReq.MaxResults, "max-results", getReq.MaxResults, `Specifies the maximum number of privileges to return (page length).`)
	cmd.Flags().StringVar(&getReq.PageToken, "page-token", getReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)
	cmd.Flags().StringVar(&getReq.Principal, "principal", getReq.Principal, `If provided, only the permissions for the specified principal (user or group) are returned.`)

	cmd.Use = "get SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Get permissions.`
	cmd.Long = `Get permissions.
  
  Gets the permissions for a securable. Does not include inherited permissions.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.SecurableType = args[0]
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

	cmd.Flags().IntVar(&getEffectiveReq.MaxResults, "max-results", getEffectiveReq.MaxResults, `Specifies the maximum number of privileges to return (page length).`)
	cmd.Flags().StringVar(&getEffectiveReq.PageToken, "page-token", getEffectiveReq.PageToken, `Opaque token for the next page of results (pagination).`)
	cmd.Flags().StringVar(&getEffectiveReq.Principal, "principal", getEffectiveReq.Principal, `If provided, only the effective permissions for the specified principal (user or group) are returned.`)

	cmd.Use = "get-effective SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Get effective permissions.`
	cmd.Long = `Get effective permissions.
  
  Gets the effective permissions for a securable. Includes inherited permissions
  from any parent securables.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getEffectiveReq.SecurableType = args[0]
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
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.SecurableType = args[0]
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
