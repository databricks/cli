// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package permission_migration

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission-migration",
		Short: `APIs for migrating acl permissions, used only by the ucx tool: https://github.com/databrickslabs/ucx.`,
		Long: `APIs for migrating acl permissions, used only by the ucx tool:
  https://github.com/databrickslabs/ucx`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newMigratePermissions())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start migrate-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var migratePermissionsOverrides []func(
	*cobra.Command,
	*iam.MigratePermissionsRequest,
)

func newMigratePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var migratePermissionsReq iam.MigratePermissionsRequest
	var migratePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&migratePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&migratePermissionsReq.Size, "size", migratePermissionsReq.Size, `The maximum number of permissions that will be migrated.`)

	cmd.Use = "migrate-permissions WORKSPACE_ID FROM_WORKSPACE_GROUP_NAME TO_ACCOUNT_GROUP_NAME"
	cmd.Short = `Migrate Permissions.`
	cmd.Long = `Migrate Permissions.

  Arguments:
    WORKSPACE_ID: WorkspaceId of the associated workspace where the permission migration
      will occur.
    FROM_WORKSPACE_GROUP_NAME: The name of the workspace group that permissions will be migrated from.
    TO_ACCOUNT_GROUP_NAME: The name of the account group that permissions will be migrated to.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'workspace_id', 'from_workspace_group_name', 'to_account_group_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := migratePermissionsJson.Unmarshal(&migratePermissionsReq)
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
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[0], &migratePermissionsReq.WorkspaceId)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
			}
		}
		if !cmd.Flags().Changed("json") {
			migratePermissionsReq.FromWorkspaceGroupName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			migratePermissionsReq.ToAccountGroupName = args[2]
		}

		response, err := w.PermissionMigration.MigratePermissions(ctx, migratePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range migratePermissionsOverrides {
		fn(cmd, &migratePermissionsReq)
	}

	return cmd
}

// end service PermissionMigration
