// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings_v2

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
		Use:     "settings-v2",
		Short:   `APIs to manage account level settings.`,
		Long:    `APIs to manage account level settings`,
		GroupID: "settings",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGetPublicAccountSetting())
	cmd.AddCommand(newListAccountSettingsMetadata())
	cmd.AddCommand(newPatchPublicAccountSetting())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-public-account-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPublicAccountSettingOverrides []func(
	*cobra.Command,
	*settingsv2.GetPublicAccountSettingRequest,
)

func newGetPublicAccountSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var getPublicAccountSettingReq settingsv2.GetPublicAccountSettingRequest

	cmd.Use = "get-public-account-setting NAME"
	cmd.Short = `Get an account setting.`
	cmd.Long = `Get an account setting.

  Get a setting value at account level. See
  :method:settingsv2/listaccountsettingsmetadata for list of setting available
  via public APIs at account level.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getPublicAccountSettingReq.Name = args[0]

		response, err := a.SettingsV2.GetPublicAccountSetting(ctx, getPublicAccountSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPublicAccountSettingOverrides {
		fn(cmd, &getPublicAccountSettingReq)
	}

	return cmd
}

// start list-account-settings-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listAccountSettingsMetadataOverrides []func(
	*cobra.Command,
	*settingsv2.ListAccountSettingsMetadataRequest,
)

func newListAccountSettingsMetadata() *cobra.Command {
	cmd := &cobra.Command{}

	var listAccountSettingsMetadataReq settingsv2.ListAccountSettingsMetadataRequest

	cmd.Flags().IntVar(&listAccountSettingsMetadataReq.PageSize, "page-size", listAccountSettingsMetadataReq.PageSize, `The maximum number of settings to return.`)
	cmd.Flags().StringVar(&listAccountSettingsMetadataReq.PageToken, "page-token", listAccountSettingsMetadataReq.PageToken, `A page token, received from a previous ListAccountSettingsMetadataRequest call.`)

	cmd.Use = "list-account-settings-metadata"
	cmd.Short = `List valid setting keys and their metadata.`
	cmd.Long = `List valid setting keys and their metadata.

  List valid setting keys and metadata. These settings are available to be
  referenced via GET :method:settingsv2/getpublicaccountsetting and PATCH
  :method:settingsv2/patchpublicaccountsetting APIs`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response := a.SettingsV2.ListAccountSettingsMetadata(ctx, listAccountSettingsMetadataReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listAccountSettingsMetadataOverrides {
		fn(cmd, &listAccountSettingsMetadataReq)
	}

	return cmd
}

// start patch-public-account-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchPublicAccountSettingOverrides []func(
	*cobra.Command,
	*settingsv2.PatchPublicAccountSettingRequest,
)

func newPatchPublicAccountSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var patchPublicAccountSettingReq settingsv2.PatchPublicAccountSettingRequest
	patchPublicAccountSettingReq.Setting = settingsv2.Setting{}
	var patchPublicAccountSettingJson flags.JsonFlag

	cmd.Flags().Var(&patchPublicAccountSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
	cmd.Flags().StringVar(&patchPublicAccountSettingReq.Setting.Name, "name", patchPublicAccountSettingReq.Setting.Name, `Name of the setting.`)
	// TODO: complex arg: personal_compute
	// TODO: complex arg: restrict_workspace_admins
	// TODO: complex arg: string_val

	cmd.Use = "patch-public-account-setting NAME"
	cmd.Short = `Update an account setting.`
	cmd.Long = `Update an account setting.

  Patch a setting value at account level. See
  :method:settingsv2/listaccountsettingsmetadata for list of setting available
  via public APIs at account level. To determine the correct field to include in
  a patch request, refer to the type field of the setting returned in the
  :method:settingsv2/listaccountsettingsmetadata response.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := patchPublicAccountSettingJson.Unmarshal(&patchPublicAccountSettingReq.Setting)
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
		patchPublicAccountSettingReq.Name = args[0]

		response, err := a.SettingsV2.PatchPublicAccountSetting(ctx, patchPublicAccountSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchPublicAccountSettingOverrides {
		fn(cmd, &patchPublicAccountSettingReq)
	}

	return cmd
}

// end service AccountSettingsV2
