// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings_v2

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
		Use:     "settings-v2",
		Short:   `APIs to manage account level settings.`,
		Long:    `APIs to manage account level settings`,
		GroupID: "settings",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newGetPublicAccountSetting())
	cmd.AddCommand(newGetPublicAccountUserPreference())
	cmd.AddCommand(newListAccountSettingsMetadata())
	cmd.AddCommand(newListAccountUserPreferencesMetadata())
	cmd.AddCommand(newPatchPublicAccountSetting())
	cmd.AddCommand(newPatchPublicAccountUserPreference())

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
	*sdksettings.GetPublicAccountSettingRequest,
)

func newGetPublicAccountSetting() *cobra.Command {
	cmd := &cobra.Command{}
	var getPublicAccountSettingReq sdksettings.GetPublicAccountSettingRequest

	cmd.Use = "get-public-account-setting NAME"
	cmd.Short = `Get an account setting.`
	cmd.Long = `Get an account setting.

  Get a setting value at account level. See
  :method:settingsv2/listaccountsettingsmetadata for list of setting available
  via public APIs at account level.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getPublicAccountSettingReq.Name = new(args[0])

		sdkReq := &getPublicAccountSettingReq
		response, err := a.Settings.GetPublicAccountSetting(ctx, *sdkReq)
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
	for _, fn := range getPublicAccountSettingOverrides {
		fn(cmd, &getPublicAccountSettingReq)
	}

	return cmd
}

// start get-public-account-user-preference command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPublicAccountUserPreferenceOverrides []func(
	*cobra.Command,
	*sdksettings.GetPublicAccountUserPreferenceRequest,
)

func newGetPublicAccountUserPreference() *cobra.Command {
	cmd := &cobra.Command{}
	var getPublicAccountUserPreferenceReq sdksettings.GetPublicAccountUserPreferenceRequest

	cmd.Use = "get-public-account-user-preference USER_ID NAME"
	cmd.Short = `*Beta* Get a user preference.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a user preference.

  Get a user preference for a specific user. User preferences are personal
  settings that allow individual customization without affecting other users.
  See :method:settingsv2/listaccountuserpreferencesmetadata for list of user
  preferences available via public APIs.

  Arguments:
    USER_ID: User ID of the user whose setting is being retrieved.
    NAME: User Setting name.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getPublicAccountUserPreferenceReq.UserId = new(args[0])
		getPublicAccountUserPreferenceReq.Name = new(args[1])

		sdkReq := &getPublicAccountUserPreferenceReq
		response, err := a.Settings.GetPublicAccountUserPreference(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := settings.UserPreferenceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPublicAccountUserPreferenceOverrides {
		fn(cmd, &getPublicAccountUserPreferenceReq)
	}

	return cmd
}

// start list-account-settings-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listAccountSettingsMetadataOverrides []func(
	*cobra.Command,
	*settings.ListAccountSettingsMetadataRequest,
)

func newListAccountSettingsMetadata() *cobra.Command {
	cmd := &cobra.Command{}
	var listAccountSettingsMetadataReq settings.ListAccountSettingsMetadataRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listAccountSettingsMetadataLimit int

	flags.IntPtrVar(cmd.Flags(), &listAccountSettingsMetadataReq.PageSize, "page-size", `The maximum number of settings to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listAccountSettingsMetadataLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listAccountSettingsMetadataReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-account-settings-metadata"
	cmd.Short = `List valid setting keys and their metadata.`
	cmd.Long = `List valid setting keys and their metadata.

  List valid setting keys and metadata. These settings are available to be
  referenced via GET :method:settingsv2/getpublicaccountsetting and PATCH
  :method:settingsv2/patchpublicaccountsetting APIs`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq, err := settings.ListAccountSettingsMetadataRequestToSDK(&listAccountSettingsMetadataReq)
		if err != nil {
			return err
		}
		if listAccountSettingsMetadataLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listAccountSettingsMetadataLimit)
		}
		if listAccountSettingsMetadataLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listAccountSettingsMetadataLimit)
		}
		response := a.Settings.ListAccountSettingsMetadataIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, settings.SettingsMetadataFromSDK))
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

// start list-account-user-preferences-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listAccountUserPreferencesMetadataOverrides []func(
	*cobra.Command,
	*settings.ListAccountUserPreferencesMetadataRequest,
)

func newListAccountUserPreferencesMetadata() *cobra.Command {
	cmd := &cobra.Command{}
	var listAccountUserPreferencesMetadataReq settings.ListAccountUserPreferencesMetadataRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listAccountUserPreferencesMetadataLimit int

	flags.IntPtrVar(cmd.Flags(), &listAccountUserPreferencesMetadataReq.PageSize, "page-size", `The maximum number of settings to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listAccountUserPreferencesMetadataLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listAccountUserPreferencesMetadataReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-account-user-preferences-metadata USER_ID"
	cmd.Short = `*Beta* List user preferences and their metadata.`
	cmd.Long = `This command is in Beta and may change without notice.

List user preferences and their metadata.

  List valid user preferences and their metadata for a specific user. User
  preferences are personal settings that allow individual customization without
  affecting other users. These settings are available to be referenced via GET
  :method:settingsv2/getpublicaccountuserpreference and PATCH
  :method:settingsv2/patchpublicaccountuserpreference APIs

  Arguments:
    USER_ID: User ID of the user whose settings metadata is being retrieved.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		listAccountUserPreferencesMetadataReq.UserId = new(args[0])

		sdkReq, err := settings.ListAccountUserPreferencesMetadataRequestToSDK(&listAccountUserPreferencesMetadataReq)
		if err != nil {
			return err
		}
		if listAccountUserPreferencesMetadataLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listAccountUserPreferencesMetadataLimit)
		}
		if listAccountUserPreferencesMetadataLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listAccountUserPreferencesMetadataLimit)
		}
		response := a.Settings.ListAccountUserPreferencesMetadataIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, settings.SettingsMetadataFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listAccountUserPreferencesMetadataOverrides {
		fn(cmd, &listAccountUserPreferencesMetadataReq)
	}

	return cmd
}

// start patch-public-account-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchPublicAccountSettingOverrides []func(
	*cobra.Command,
	*settings.PatchPublicAccountSettingRequest,
)

func newPatchPublicAccountSetting() *cobra.Command {
	cmd := &cobra.Command{}
	var patchPublicAccountSettingReq settings.PatchPublicAccountSettingRequest
	patchPublicAccountSettingReq.Setting = &settings.Setting{}

	var patchPublicAccountSettingJson flags.JsonFlag

	cmd.Flags().Var(&patchPublicAccountSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
	flags.StringPtrVar(cmd.Flags(), &patchPublicAccountSettingReq.Name, "name", `Name of the setting.`)
	// TODO: complex arg: operational_email_custom_recipient
	// TODO: complex arg: personal_compute
	// TODO: complex arg: restrict_workspace_admins
	// TODO: complex arg: string_val
	// TODO: complex arg: workspace_label

	cmd.Use = "patch-public-account-setting NAME"
	cmd.Short = `Update an account setting.`
	cmd.Long = `Update an account setting.

  Patch a setting value at account level. See
  :method:settingsv2/listaccountsettingsmetadata for list of setting available
  via public APIs at account level. To determine the correct field to include in
  a patch request, refer to the type field of the setting returned in the
  :method:settingsv2/listaccountsettingsmetadata response.

  Note: Page refresh is required for changes to take effect in UI.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := patchPublicAccountSettingJson.Unmarshal(&patchPublicAccountSettingReq.Setting)
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
		patchPublicAccountSettingReq.Name = new(args[0])

		sdkReq, err := settings.PatchPublicAccountSettingRequestToSDK(&patchPublicAccountSettingReq)
		if err != nil {
			return err
		}
		response, err := a.Settings.PatchPublicAccountSetting(ctx, *sdkReq)
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
	for _, fn := range patchPublicAccountSettingOverrides {
		fn(cmd, &patchPublicAccountSettingReq)
	}

	return cmd
}

// start patch-public-account-user-preference command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchPublicAccountUserPreferenceOverrides []func(
	*cobra.Command,
	*settings.PatchPublicAccountUserPreferenceRequest,
)

func newPatchPublicAccountUserPreference() *cobra.Command {
	cmd := &cobra.Command{}
	var patchPublicAccountUserPreferenceReq settings.PatchPublicAccountUserPreferenceRequest
	patchPublicAccountUserPreferenceReq.Setting = &settings.UserPreference{}

	var patchPublicAccountUserPreferenceJson flags.JsonFlag

	cmd.Flags().Var(&patchPublicAccountUserPreferenceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: boolean_val
	// TODO: complex arg: effective_boolean_val
	// TODO: complex arg: effective_string_val
	flags.StringPtrVar(cmd.Flags(), &patchPublicAccountUserPreferenceReq.Name, "name", `Name of the setting.`)
	// TODO: complex arg: string_val
	flags.StringPtrVar(cmd.Flags(), &patchPublicAccountUserPreferenceReq.UserId, "user-id", `User ID of the user.`)

	cmd.Use = "patch-public-account-user-preference USER_ID NAME"
	cmd.Short = `*Beta* Update a user preference.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a user preference.

  Update a user preference for a specific user. User preferences are personal
  settings that allow individual customization without affecting other users.
  See :method:settingsv2/listaccountuserpreferencesmetadata for list of user
  preferences available via public APIs.

  Note: Page refresh is required for changes to take effect in UI.

  Arguments:
    USER_ID: User ID of the user whose setting is being updated.
    NAME: `

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := patchPublicAccountUserPreferenceJson.Unmarshal(&patchPublicAccountUserPreferenceReq.Setting)
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
		patchPublicAccountUserPreferenceReq.UserId = new(args[0])
		patchPublicAccountUserPreferenceReq.Name = new(args[1])

		sdkReq, err := settings.PatchPublicAccountUserPreferenceRequestToSDK(&patchPublicAccountUserPreferenceReq)
		if err != nil {
			return err
		}
		response, err := a.Settings.PatchPublicAccountUserPreference(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := settings.UserPreferenceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchPublicAccountUserPreferenceOverrides {
		fn(cmd, &patchPublicAccountUserPreferenceReq)
	}

	return cmd
}

// end service AccountSettingsV2
