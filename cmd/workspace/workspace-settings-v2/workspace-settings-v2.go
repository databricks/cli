// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_settings_v2

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settingsv2"
	"github.com/spf13/cobra"
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
	*settingsv2.GetPublicWorkspaceSettingRequest,
)

func newGetPublicWorkspaceSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var getPublicWorkspaceSettingReq settingsv2.GetPublicWorkspaceSettingRequest

	cmd.Use = "get-public-workspace-setting NAME"
	cmd.Short = `Get a workspace setting.`
	cmd.Long = `Get a workspace setting.

  Get a setting value at workspace level. See
  :method:settingsv2/listworkspacesettingsmetadata for list of setting available
  via public APIs.

  Arguments:
    NAME: Name of the setting`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPublicWorkspaceSettingReq.Name = args[0]

		response, err := w.WorkspaceSettingsV2.GetPublicWorkspaceSetting(ctx, getPublicWorkspaceSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*settingsv2.ListWorkspaceSettingsMetadataRequest,
)

func newListWorkspaceSettingsMetadata() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceSettingsMetadataReq settingsv2.ListWorkspaceSettingsMetadataRequest

	cmd.Flags().IntVar(&listWorkspaceSettingsMetadataReq.PageSize, "page-size", listWorkspaceSettingsMetadataReq.PageSize, `The maximum number of settings to return.`)
	cmd.Flags().StringVar(&listWorkspaceSettingsMetadataReq.PageToken, "page-token", listWorkspaceSettingsMetadataReq.PageToken, `A page token, received from a previous ListWorkspaceSettingsMetadataRequest call.`)

	cmd.Use = "list-workspace-settings-metadata"
	cmd.Short = `List valid setting keys and their metadata.`
	cmd.Long = `List valid setting keys and their metadata.

  List valid setting keys and metadata. These settings are available to be
  referenced via GET :method:settingsv2/getpublicworkspacesetting and PATCH
  :method:settingsv2/patchpublicworkspacesetting APIs`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.WorkspaceSettingsV2.ListWorkspaceSettingsMetadata(ctx, listWorkspaceSettingsMetadataReq)
		return cmdio.RenderIterator(ctx, response)
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
	*settingsv2.PatchPublicWorkspaceSettingRequest,
)

func newPatchPublicWorkspaceSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var patchPublicWorkspaceSettingReq settingsv2.PatchPublicWorkspaceSettingRequest
	patchPublicWorkspaceSettingReq.Setting = settingsv2.Setting{}
	var patchPublicWorkspaceSettingJson flags.JsonFlag

	cmd.Flags().Var(&patchPublicWorkspaceSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aibi_dashboard_embedding_access_policy
	// TODO: complex arg: aibi_dashboard_embedding_approved_domains
	// TODO: complex arg: automatic_cluster_update_workspace
	// TODO: complex arg: boolean_val
	// TODO: complex arg: effective_aibi_dashboard_embedding_access_policy
	// TODO: complex arg: effective_aibi_dashboard_embedding_approved_domains
	// TODO: complex arg: effective_automatic_cluster_update_workspace
	// TODO: complex arg: effective_boolean_val
	// TODO: complex arg: effective_integer_val
	// TODO: complex arg: effective_personal_compute
	// TODO: complex arg: effective_restrict_workspace_admins
	// TODO: complex arg: effective_string_val
	// TODO: complex arg: integer_val
	cmd.Flags().StringVar(&patchPublicWorkspaceSettingReq.Setting.Name, "name", patchPublicWorkspaceSettingReq.Setting.Name, `Name of the setting.`)
	// TODO: complex arg: personal_compute
	// TODO: complex arg: restrict_workspace_admins
	// TODO: complex arg: string_val

	cmd.Use = "patch-public-workspace-setting NAME"
	cmd.Short = `Update a workspace setting.`
	cmd.Long = `Update a workspace setting.

  Patch a setting value at workspace level. See
  :method:settingsv2/listworkspacesettingsmetadata for list of setting available
  via public APIs at workspace level. To determine the correct field to include
  in a patch request, refer to the type field of the setting returned in the
  :method:settingsv2/listworkspacesettingsmetadata response.

  Arguments:
    NAME: Name of the setting`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := patchPublicWorkspaceSettingJson.Unmarshal(&patchPublicWorkspaceSettingReq.Setting)
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
		patchPublicWorkspaceSettingReq.Name = args[0]

		response, err := w.WorkspaceSettingsV2.PatchPublicWorkspaceSetting(ctx, patchPublicWorkspaceSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
