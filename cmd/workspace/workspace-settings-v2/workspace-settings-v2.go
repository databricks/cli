// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_settings_v2

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	settings "github.com/databricks/cli/libs/sdktypes/settings/v2"
	sdksettings "github.com/databricks/sdk-go/settings/v2"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace-settings-v2",
		Short:   `APIs to manage workspace level settings.`,
		Long:    `APIs to manage workspace level settings`,
		GroupID: "settings",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newGetPublicWorkspaceSetting())
	cmd.AddCommand(newListWorkspaceSettingsMetadata())
	cmd.AddCommand(newPatchPublicWorkspaceSetting())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-public-workspace-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPublicWorkspaceSettingOverrides []func(
	*cobra.Command,
	*sdksettings.GetPublicWorkspaceSettingRequest,
)

func newGetPublicWorkspaceSetting() *cobra.Command {
	cmd := &cobra.Command{}
	var getPublicWorkspaceSettingReq sdksettings.GetPublicWorkspaceSettingRequest

	cmd.Use = "get-public-workspace-setting NAME"
	cmd.Short = `Get a workspace setting.`
	cmd.Long = `Get a workspace setting.

  Get a setting value at workspace level. See
  :method:settingsv2/listworkspacesettingsmetadata for list of setting available
  via public APIs.

  Arguments:
    NAME: Name of the setting`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getPublicWorkspaceSettingReq.Name = new(args[0])

		sdkReq := &getPublicWorkspaceSettingReq
		response, err := w.Settings.GetPublicWorkspaceSetting(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := settings.SettingFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPublicWorkspaceSettingOverrides {
		fn(cmd, &getPublicWorkspaceSettingReq)
	}

	return cmd
}

// start list-workspace-settings-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceSettingsMetadataOverrides []func(
	*cobra.Command,
	*settings.ListWorkspaceSettingsMetadataRequest,
)

func newListWorkspaceSettingsMetadata() *cobra.Command {
	cmd := &cobra.Command{}
	var listWorkspaceSettingsMetadataReq settings.ListWorkspaceSettingsMetadataRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceSettingsMetadataLimit int

	flags.IntPtrVar(cmd.Flags(), &listWorkspaceSettingsMetadataReq.PageSize, "page-size", `The maximum number of settings to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceSettingsMetadataLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listWorkspaceSettingsMetadataReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-workspace-settings-metadata"
	cmd.Short = `List valid setting keys and their metadata.`
	cmd.Long = `List valid setting keys and their metadata.

  List valid setting keys and metadata. These settings are available to be
  referenced via GET :method:settingsv2/getpublicworkspacesetting and PATCH
  :method:settingsv2/patchpublicworkspacesetting APIs`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := settings.ListWorkspaceSettingsMetadataRequestToSDK(&listWorkspaceSettingsMetadataReq)
		if err != nil {
			return err
		}
		if listWorkspaceSettingsMetadataLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceSettingsMetadataLimit)
		}
		if listWorkspaceSettingsMetadataLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listWorkspaceSettingsMetadataLimit)
		}
		response := w.Settings.ListWorkspaceSettingsMetadataIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, settings.SettingsMetadataFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWorkspaceSettingsMetadataOverrides {
		fn(cmd, &listWorkspaceSettingsMetadataReq)
	}

	return cmd
}

// start patch-public-workspace-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchPublicWorkspaceSettingOverrides []func(
	*cobra.Command,
	*settings.PatchPublicWorkspaceSettingRequest,
)

func newPatchPublicWorkspaceSetting() *cobra.Command {
	cmd := &cobra.Command{}
	var patchPublicWorkspaceSettingReq settings.PatchPublicWorkspaceSettingRequest
	patchPublicWorkspaceSettingReq.Setting = &settings.Setting{}

	var patchPublicWorkspaceSettingJson flags.JsonFlag

	cmd.Flags().Var(&patchPublicWorkspaceSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aibi_dashboard_embedding_access_policy
	// TODO: complex arg: aibi_dashboard_embedding_approved_domains
	// TODO: complex arg: allowed_apps_user_api_scopes
	// TODO: complex arg: automatic_cluster_update_workspace
	// TODO: complex arg: boolean_val
	// TODO: complex arg: collaboration_platform_connectivity
	// TODO: complex arg: effective_aibi_dashboard_embedding_access_policy
	// TODO: complex arg: effective_aibi_dashboard_embedding_approved_domains
	// TODO: complex arg: effective_allowed_apps_user_api_scopes
	// TODO: complex arg: effective_automatic_cluster_update_workspace
	// TODO: complex arg: effective_boolean_val
	// TODO: complex arg: effective_collaboration_platform_connectivity
	// TODO: complex arg: effective_integer_val
	// TODO: complex arg: effective_operational_email_custom_recipient
	// TODO: complex arg: effective_personal_compute
	// TODO: complex arg: effective_restrict_workspace_admins
	// TODO: complex arg: effective_string_val
	// TODO: complex arg: effective_workspace_label
	// TODO: complex arg: integer_val
	flags.StringPtrVar(cmd.Flags(), &patchPublicWorkspaceSettingReq.Name, "name", `Name of the setting.`)
	// TODO: complex arg: operational_email_custom_recipient
	// TODO: complex arg: personal_compute
	// TODO: complex arg: restrict_workspace_admins
	// TODO: complex arg: string_val
	// TODO: complex arg: workspace_label

	cmd.Use = "patch-public-workspace-setting NAME"
	cmd.Short = `Update a workspace setting.`
	cmd.Long = `Update a workspace setting.

  Patch a setting value at workspace level. See
  :method:settingsv2/listworkspacesettingsmetadata for list of setting available
  via public APIs at workspace level. To determine the correct field to include
  in a patch request, refer to the type field of the setting returned in the
  :method:settingsv2/listworkspacesettingsmetadata response.

  Note: Page refresh is required for changes to take effect in UI.

  Arguments:
    NAME: Name of the setting`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := patchPublicWorkspaceSettingJson.Unmarshal(&patchPublicWorkspaceSettingReq.Setting)
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
		patchPublicWorkspaceSettingReq.Name = new(args[0])

		sdkReq, err := settings.PatchPublicWorkspaceSettingRequestToSDK(&patchPublicWorkspaceSettingReq)
		if err != nil {
			return err
		}
		response, err := w.Settings.PatchPublicWorkspaceSetting(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := settings.SettingFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchPublicWorkspaceSettingOverrides {
		fn(cmd, &patchPublicWorkspaceSettingReq)
	}

	return cmd
}

// end service WorkspaceSettingsV2
