// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package permission_migration

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
	*map[string]any,
)

func newMigratePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var migratePermissionsReq map[string]any
	migratePermissionsReq = map[string]any{}
	var migratePermissionsFromWorkspaceGroupNameRaw string
	var migratePermissionsSizeRaw int
	var migratePermissionsToAccountGroupNameRaw string
	var migratePermissionsWorkspaceIdRaw int64

	var migratePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&migratePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&migratePermissionsSizeRaw, "size", migratePermissionsSizeRaw, `The maximum number of permissions that will be migrated.`)

	cmd.Use = "migrate-permissions WORKSPACE_ID FROM_WORKSPACE_GROUP_NAME TO_ACCOUNT_GROUP_NAME"
	cmd.Short = `Migrate Permissions.`
	cmd.Long = `Migrate Permissions.

  Arguments:
    WORKSPACE_ID: WorkspaceId of the associated workspace where the permission migration
      will occur.
    FROM_WORKSPACE_GROUP_NAME: The name of the workspace group that permissions will be migrated from.
    TO_ACCOUNT_GROUP_NAME: The name of the account group that permissions will be migrated to.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'workspace_id', 'from_workspace_group_name', 'to_account_group_name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := migratePermissionsJson.Unmarshal(&migratePermissionsReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[0], &migratePermissionsWorkspaceIdRaw)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
			}
		}
		if !cmd.Flags().Changed("json") {
			migratePermissionsFromWorkspaceGroupNameRaw = args[1]
		}
		if !cmd.Flags().Changed("json") {
			migratePermissionsToAccountGroupNameRaw = args[2]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("from-workspace-group-name") || migratePermissionsFromWorkspaceGroupNameRaw != "" {
			migratePermissionsReq["from_workspace_group_name"] = migratePermissionsFromWorkspaceGroupNameRaw
		}
		if cmd.Flags().Changed("size") || migratePermissionsSizeRaw != 0 {
			migratePermissionsReq["size"] = migratePermissionsSizeRaw
		}
		if cmd.Flags().Changed("to-account-group-name") || migratePermissionsToAccountGroupNameRaw != "" {
			migratePermissionsReq["to_account_group_name"] = migratePermissionsToAccountGroupNameRaw
		}
		if cmd.Flags().Changed("workspace-id") || migratePermissionsWorkspaceIdRaw != 0 {
			migratePermissionsReq["workspace_id"] = migratePermissionsWorkspaceIdRaw
		}
		rawPath := "/api/2.0/permissionmigration"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, migratePermissionsReq, &response)
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
